package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type dataPipelineStore struct {
	mu sync.Mutex

	nextPipelineID int64
	nextObjectID   int64
	nextTaskID     int64

	uniqueIDToPipelineID map[string]string
	pipelines            map[string]*dataPipelinePipeline
	definitions          map[string]*dataPipelineDefinition
	tasks                map[string]*dataPipelineTask
	tags                 map[string]map[string]string
}

type dataPipelinePipeline struct {
	ID            string
	Name          string
	UniqueID      string
	Description   string
	State         string
	CreatedAt     string
	UpdatedAt     string
	ActivatedAt   string
	DeactivatedAt string
}

type dataPipelineDefinition struct {
	PipelineObjects  []map[string]any
	ParameterObjects []map[string]any
	ParameterValues  []map[string]any
}

type dataPipelineTask struct {
	TaskID        string
	PipelineID    string
	AttemptID     string
	WorkerGroup   string
	Status        string
	LastHeartbeat string
	LastProgress  string
	ObjectID      string
}

func newDataPipelineStore() *dataPipelineStore {
	s := &dataPipelineStore{
		nextPipelineID:       2,
		nextObjectID:         2,
		nextTaskID:           2,
		uniqueIDToPipelineID: map[string]string{},
		pipelines:            map[string]*dataPipelinePipeline{},
		definitions:          map[string]*dataPipelineDefinition{},
		tasks:                map[string]*dataPipelineTask{},
		tags:                 map[string]map[string]string{},
	}
	s.seedLocked(time.Now().UTC())
	return s
}

func (s *dataPipelineStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	pipelineID := dataPipelinePayloadString(payload, []string{"pipelineId", "PipelineId"}, s.firstPipelineIDLocked())
	if pipelineID == "" {
		pipelineID = "df-000001"
	}

	switch action {
	case "CreatePipeline":
		name := dataPipelinePayloadString(payload, []string{"name", "Name"}, "stackyard-pipeline")
		uniqueID := dataPipelinePayloadString(payload, []string{"uniqueId", "UniqueId"}, "stackyard-unique")
		if existing := s.uniqueIDToPipelineID[uniqueID]; existing != "" {
			return map[string]any{"pipelineId": existing}
		}

		id := dataPipelinePayloadString(payload, []string{"pipelineId", "PipelineId"}, "")
		if id == "" {
			id = fmt.Sprintf("df-%06d", s.nextPipelineID)
			s.nextPipelineID++
		}
		if !strings.HasPrefix(id, "df-") {
			id = fmt.Sprintf("df-%06d", s.nextPipelineID)
			s.nextPipelineID++
		}

		p := &dataPipelinePipeline{
			ID:            id,
			Name:          name,
			UniqueID:      uniqueID,
			Description:   dataPipelinePayloadString(payload, []string{"description", "Description"}, "Stackyard Data Pipeline"),
			State:         "INACTIVE",
			CreatedAt:     now,
			UpdatedAt:     now,
			ActivatedAt:   "",
			DeactivatedAt: "",
		}
		s.pipelines[id] = p
		s.uniqueIDToPipelineID[uniqueID] = id
		s.ensureDefinitionLocked(id)
		s.ensureTaskLocked(id, now)
		s.ensureTagsLocked(id)
		return map[string]any{"pipelineId": id}

	case "DeletePipeline":
		p := s.ensurePipelineLocked(pipelineID, now)
		delete(s.uniqueIDToPipelineID, p.UniqueID)
		delete(s.pipelines, pipelineID)
		delete(s.definitions, pipelineID)
		delete(s.tasks, pipelineID)
		delete(s.tags, pipelineID)
		return map[string]any{}

	case "ActivatePipeline":
		p := s.ensurePipelineLocked(pipelineID, now)
		p.State = "ACTIVE"
		p.ActivatedAt = now
		p.UpdatedAt = now
		return map[string]any{}

	case "DeactivatePipeline":
		p := s.ensurePipelineLocked(pipelineID, now)
		p.State = "INACTIVE"
		p.DeactivatedAt = now
		p.UpdatedAt = now
		return map[string]any{}

	case "AddTags":
		tagMap := s.ensureTagsLocked(pipelineID)
		for k, v := range dataPipelineExtractTags(payload) {
			tagMap[k] = v
		}
		return map[string]any{}

	case "RemoveTags":
		tagMap := s.ensureTagsLocked(pipelineID)
		for _, key := range dataPipelinePayloadStringSlice(payload, "tagKeys") {
			delete(tagMap, key)
		}
		return map[string]any{}

	case "ListPipelines":
		items := make([]any, 0, len(s.pipelines))
		for _, id := range s.sortedPipelineIDsLocked() {
			p := s.ensurePipelineLocked(id, now)
			items = append(items, map[string]any{
				"id":   p.ID,
				"name": p.Name,
			})
		}
		return map[string]any{
			"pipelineIdList": items,
			"hasMoreResults": false,
		}

	case "DescribePipelines":
		ids := dataPipelinePayloadStringSlice(payload, "pipelineIds")
		if len(ids) == 0 {
			ids = s.sortedPipelineIDsLocked()
		}
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			p := s.ensurePipelineLocked(strings.TrimSpace(id), now)
			fields := []any{
				map[string]any{"key": "@pipelineState", "stringValue": p.State},
				map[string]any{"key": "@creationTime", "stringValue": p.CreatedAt},
				map[string]any{"key": "@description", "stringValue": p.Description},
			}
			out = append(out, map[string]any{
				"pipelineId": p.ID,
				"name":       p.Name,
				"fields":     fields,
				"tags":       dataPipelineTagList(s.ensureTagsLocked(p.ID)),
			})
		}
		return map[string]any{
			"pipelineDescriptionList": out,
			"hasMoreResults":          false,
		}

	case "PutPipelineDefinition":
		p := s.ensurePipelineLocked(pipelineID, now)
		def := s.ensureDefinitionLocked(p.ID)

		if objects := dataPipelineObjectsFromPayload(payload, "pipelineObjects"); len(objects) > 0 {
			def.PipelineObjects = objects
		}
		if params := dataPipelineObjectsFromPayload(payload, "parameterObjects"); len(params) > 0 {
			def.ParameterObjects = params
		}
		if values := dataPipelineObjectsFromPayload(payload, "parameterValues"); len(values) > 0 {
			def.ParameterValues = values
		}
		p.UpdatedAt = now
		return map[string]any{
			"errored":            false,
			"validationErrors":   []any{},
			"validationWarnings": []any{},
		}

	case "GetPipelineDefinition":
		def := s.ensureDefinitionLocked(pipelineID)
		return map[string]any{
			"pipelineObjects":  dataPipelineCloneObjectList(def.PipelineObjects),
			"parameterObjects": dataPipelineCloneObjectList(def.ParameterObjects),
			"parameterValues":  dataPipelineCloneObjectList(def.ParameterValues),
		}

	case "ValidatePipelineDefinition":
		def := s.ensureDefinitionLocked(pipelineID)
		warnings := []any{}
		if len(def.PipelineObjects) == 0 {
			warnings = append(warnings, map[string]any{
				"id":       "pipeline",
				"warnings": []any{"pipelineObjects is empty; using default definition"},
			})
		}
		return map[string]any{
			"errored":            false,
			"validationErrors":   []any{},
			"validationWarnings": warnings,
		}

	case "QueryObjects":
		def := s.ensureDefinitionLocked(pipelineID)
		ids := make([]any, 0, len(def.PipelineObjects))
		for _, obj := range def.PipelineObjects {
			id := dataPipelinePayloadString(obj, []string{"id", "Id"}, "")
			if id != "" {
				ids = append(ids, id)
			}
		}
		return map[string]any{
			"ids":            ids,
			"hasMoreResults": false,
			"marker":         "",
		}

	case "DescribeObjects":
		def := s.ensureDefinitionLocked(pipelineID)
		requestedIDs := dataPipelinePayloadStringSlice(payload, "objectIds")
		if len(requestedIDs) == 0 {
			for _, obj := range def.PipelineObjects {
				if id := dataPipelinePayloadString(obj, []string{"id", "Id"}, ""); id != "" {
					requestedIDs = append(requestedIDs, id)
				}
			}
		}

		out := make([]any, 0, len(requestedIDs))
		for _, requestedID := range requestedIDs {
			obj := dataPipelineFindPipelineObject(def.PipelineObjects, requestedID)
			if obj == nil {
				obj = map[string]any{
					"id":   requestedID,
					"name": "object-" + requestedID,
					"fields": []any{
						map[string]any{"key": "type", "stringValue": "Default"},
					},
				}
			}
			out = append(out, dataPipelineCloneObject(obj))
		}
		return map[string]any{
			"pipelineObjects": out,
			"hasMoreResults":  false,
			"marker":          "",
		}

	case "EvaluateExpression":
		expr := dataPipelinePayloadString(payload, []string{"expression", "Expression"}, "#{@scheduledStartTime}")
		return map[string]any{
			"evaluatedExpression": strings.ReplaceAll(expr, "@scheduledStartTime", now),
		}

	case "PollForTask":
		task := s.ensureTaskLocked(pipelineID, now)
		workerGroup := dataPipelinePayloadString(payload, []string{"workerGroup", "WorkerGroup"}, task.WorkerGroup)
		if strings.TrimSpace(workerGroup) == "" {
			workerGroup = "default-worker-group"
		}
		task.WorkerGroup = workerGroup
		return map[string]any{
			"taskObject": map[string]any{
				"taskId":     task.TaskID,
				"pipelineId": task.PipelineID,
				"attemptId":  task.AttemptID,
				"objects": []any{
					map[string]any{"id": "@scheduledStartTime", "stringValue": now},
					map[string]any{"id": "activityId", "stringValue": task.ObjectID},
				},
			},
		}

	case "ReportTaskRunnerHeartbeat":
		task := s.ensureTaskLocked(pipelineID, now)
		task.LastHeartbeat = now
		return map[string]any{"canceled": false}

	case "ReportTaskProgress":
		task := s.ensureTaskLocked(pipelineID, now)
		task.LastProgress = now
		return map[string]any{"canceled": false}

	case "SetTaskStatus":
		task := s.ensureTaskLocked(pipelineID, now)
		task.Status = dataPipelinePayloadString(payload, []string{"taskStatus", "TaskStatus"}, "FINISHED")
		return map[string]any{}

	case "SetStatus":
		p := s.ensurePipelineLocked(pipelineID, now)
		status := dataPipelinePayloadString(payload, []string{"status", "Status"}, "")
		if status != "" {
			p.State = status
		}
		p.UpdatedAt = now
		return map[string]any{}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"hasMoreResults": false}
	}
	if strings.HasPrefix(action, "Describe") {
		return map[string]any{"hasMoreResults": false}
	}
	return map[string]any{}
}

func (s *dataPipelineStore) seedLocked(now time.Time) {
	nowS := now.Format(time.RFC3339)
	p := &dataPipelinePipeline{
		ID:            "df-000001",
		Name:          "stackyard-seed-pipeline",
		UniqueID:      "stackyard-seed-unique",
		Description:   "Stackyard seeded pipeline",
		State:         "INACTIVE",
		CreatedAt:     nowS,
		UpdatedAt:     nowS,
		ActivatedAt:   "",
		DeactivatedAt: "",
	}
	s.pipelines[p.ID] = p
	s.uniqueIDToPipelineID[p.UniqueID] = p.ID
	s.definitions[p.ID] = &dataPipelineDefinition{
		PipelineObjects: []map[string]any{
			{
				"id":   "DefaultActivity",
				"name": "DefaultActivity",
				"fields": []any{
					map[string]any{"key": "type", "stringValue": "ShellCommandActivity"},
				},
			},
		},
		ParameterObjects: []map[string]any{
			{
				"id": "myParam",
				"attributes": []any{
					map[string]any{"key": "type", "stringValue": "String"},
				},
			},
		},
		ParameterValues: []map[string]any{
			{"id": "myParam", "stringValue": "seed"},
		},
	}
	s.tasks[p.ID] = &dataPipelineTask{
		TaskID:        "task-000001",
		PipelineID:    p.ID,
		AttemptID:     "attempt-000001",
		WorkerGroup:   "default-worker-group",
		Status:        "SCHEDULED",
		LastHeartbeat: nowS,
		LastProgress:  nowS,
		ObjectID:      "DefaultActivity",
	}
	s.tags[p.ID] = map[string]string{
		"service": "datapipeline",
	}
}

func (s *dataPipelineStore) sortedPipelineIDsLocked() []string {
	ids := make([]string, 0, len(s.pipelines))
	for id := range s.pipelines {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (s *dataPipelineStore) firstPipelineIDLocked() string {
	ids := s.sortedPipelineIDsLocked()
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func (s *dataPipelineStore) ensurePipelineLocked(pipelineID, now string) *dataPipelinePipeline {
	id := strings.TrimSpace(pipelineID)
	if id == "" {
		id = "df-000001"
	}
	if p := s.pipelines[id]; p != nil {
		return p
	}
	uniqueID := "stackyard-" + id
	p := &dataPipelinePipeline{
		ID:            id,
		Name:          "stackyard-pipeline-" + id,
		UniqueID:      uniqueID,
		Description:   "Stackyard Data Pipeline",
		State:         "INACTIVE",
		CreatedAt:     now,
		UpdatedAt:     now,
		ActivatedAt:   "",
		DeactivatedAt: "",
	}
	s.pipelines[id] = p
	s.uniqueIDToPipelineID[uniqueID] = id
	return p
}

func (s *dataPipelineStore) ensureDefinitionLocked(pipelineID string) *dataPipelineDefinition {
	id := strings.TrimSpace(pipelineID)
	if id == "" {
		id = s.firstPipelineIDLocked()
	}
	def := s.definitions[id]
	if def != nil {
		return def
	}
	objID := fmt.Sprintf("obj-%06d", s.nextObjectID)
	s.nextObjectID++
	def = &dataPipelineDefinition{
		PipelineObjects: []map[string]any{
			{
				"id":   objID,
				"name": objID,
				"fields": []any{
					map[string]any{"key": "type", "stringValue": "Default"},
				},
			},
		},
		ParameterObjects: []map[string]any{},
		ParameterValues:  []map[string]any{},
	}
	s.definitions[id] = def
	return def
}

func (s *dataPipelineStore) ensureTaskLocked(pipelineID, now string) *dataPipelineTask {
	id := strings.TrimSpace(pipelineID)
	if id == "" {
		id = s.firstPipelineIDLocked()
	}
	task := s.tasks[id]
	if task != nil {
		return task
	}
	taskID := fmt.Sprintf("task-%06d", s.nextTaskID)
	s.nextTaskID++
	task = &dataPipelineTask{
		TaskID:        taskID,
		PipelineID:    id,
		AttemptID:     fmt.Sprintf("attempt-%06d", s.nextTaskID),
		WorkerGroup:   "default-worker-group",
		Status:        "SCHEDULED",
		LastHeartbeat: now,
		LastProgress:  now,
		ObjectID:      "DefaultActivity",
	}
	s.tasks[id] = task
	return task
}

func (s *dataPipelineStore) ensureTagsLocked(pipelineID string) map[string]string {
	id := strings.TrimSpace(pipelineID)
	if id == "" {
		id = s.firstPipelineIDLocked()
	}
	tagMap := s.tags[id]
	if tagMap != nil {
		return tagMap
	}
	tagMap = map[string]string{"service": "datapipeline"}
	s.tags[id] = tagMap
	return tagMap
}

func dataPipelinePayloadString(payload map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		if v, ok := payload[key]; ok {
			switch typed := v.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					return typed
				}
			case fmt.Stringer:
				s := strings.TrimSpace(typed.String())
				if s != "" {
					return s
				}
			default:
				s := strings.TrimSpace(fmt.Sprintf("%v", v))
				if s != "" && s != "<nil>" {
					return s
				}
			}
		}
	}
	return fallback
}

func dataPipelinePayloadStringSlice(payload map[string]any, key string) []string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return out
	case string:
		s := strings.TrimSpace(typed)
		if s == "" {
			return nil
		}
		return []string{s}
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if s == "" || s == "<nil>" {
			return nil
		}
		return []string{s}
	}
}

func dataPipelineExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"tags", "Tags"} {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		switch typed := raw.(type) {
		case map[string]string:
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k != "" {
					out[k] = v
				}
			}
		case map[string]any:
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				out[k] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		case []any:
			for _, item := range typed {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				tagKey := dataPipelinePayloadString(m, []string{"key", "Key"}, "")
				if tagKey == "" {
					continue
				}
				tagValue := dataPipelinePayloadString(m, []string{"value", "Value"}, "")
				out[tagKey] = tagValue
			}
		}
	}
	return out
}

func dataPipelineTagList(tags map[string]string) []any {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{"key": k, "value": tags[k]})
	}
	return out
}

func dataPipelineObjectsFromPayload(payload map[string]any, key string) []map[string]any {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, dataPipelineCloneObject(obj))
	}
	return out
}

func dataPipelineFindPipelineObject(objects []map[string]any, objectID string) map[string]any {
	needle := strings.TrimSpace(objectID)
	for _, obj := range objects {
		if dataPipelinePayloadString(obj, []string{"id", "Id"}, "") == needle {
			return obj
		}
	}
	return nil
}

func dataPipelineCloneObjectList(in []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, dataPipelineCloneObject(item))
	}
	return out
}

func dataPipelineCloneObject(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = dataPipelineCloneAny(v)
	}
	return out
}

func dataPipelineCloneAny(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return dataPipelineCloneObject(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, dataPipelineCloneAny(item))
		}
		return out
	default:
		return typed
	}
}
