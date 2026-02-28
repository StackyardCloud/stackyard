package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type mwaaServerlessStore struct {
	mu sync.Mutex

	nextWorkflowID int64
	nextRunID      int64
	nextTaskID     int64

	workflows map[string]*mwaaServerlessWorkflow
	tags      map[string]map[string]string
}

type mwaaServerlessWorkflow struct {
	ARN         string
	Name        string
	Description string
	RoleArn     string
	Engine      float64
	TriggerMode string
	Status      string
	CreatedAt   string
	UpdatedAt   string

	LatestVersion string
	Versions      map[string]*mwaaServerlessWorkflowVersion
}

type mwaaServerlessWorkflowVersion struct {
	Version               string
	RevisionID            string
	CreatedAt             string
	UpdatedAt             string
	DefinitionS3Location  map[string]any
	LoggingConfiguration  map[string]any
	NetworkConfiguration  map[string]any
	ScheduleConfiguration map[string]any
	Runs                  map[string]*mwaaServerlessRun
}

type mwaaServerlessRun struct {
	RunID           string
	WorkflowVersion string
	Status          string
	StartedAt       string
	EndedAt         string
	Tasks           map[string]*mwaaServerlessTaskInstance
}

type mwaaServerlessTaskInstance struct {
	TaskInstanceID string
	TaskName       string
	Status         string
	StartedAt      string
	EndedAt        string
}

func newMWAAServerlessStore() *mwaaServerlessStore {
	now := time.Now().UTC().Format(time.RFC3339)
	workflowArn := "arn:aws:mwaa-serverless:us-east-1:123456789012:workflow/stackyard-workflow"

	version := &mwaaServerlessWorkflowVersion{
		Version:               "1",
		RevisionID:            "rev-000001",
		CreatedAt:             now,
		UpdatedAt:             now,
		DefinitionS3Location:  map[string]any{"Bucket": "stackyard-mwaa-serverless", "ObjectKey": "workflows/workflow.yaml", "VersionId": "1"},
		LoggingConfiguration:  map[string]any{"LogGroupName": "/aws/mwaa-serverless/stackyard-workflow"},
		NetworkConfiguration:  map[string]any{"SubnetIds": []any{"subnet-0123456789abcdef0"}, "SecurityGroupIds": []any{"sg-0123456789abcdef0"}},
		ScheduleConfiguration: map[string]any{},
		Runs:                  map[string]*mwaaServerlessRun{},
	}
	run := &mwaaServerlessRun{
		RunID:           "run-000001",
		WorkflowVersion: "1",
		Status:          "SUCCEEDED",
		StartedAt:       now,
		EndedAt:         now,
		Tasks: map[string]*mwaaServerlessTaskInstance{
			"task-000001": {
				TaskInstanceID: "task-000001",
				TaskName:       "seed-task",
				Status:         "SUCCEEDED",
				StartedAt:      now,
				EndedAt:        now,
			},
		},
	}
	version.Runs[run.RunID] = run

	workflow := &mwaaServerlessWorkflow{
		ARN:           workflowArn,
		Name:          "stackyard-workflow",
		Description:   "stackyard mwaa serverless workflow",
		RoleArn:       "arn:aws:iam::123456789012:role/stackyard-mwaa-serverless-role",
		Engine:        2.10,
		TriggerMode:   "ON_DEMAND",
		Status:        "ACTIVE",
		CreatedAt:     now,
		UpdatedAt:     now,
		LatestVersion: "1",
		Versions: map[string]*mwaaServerlessWorkflowVersion{
			version.Version: version,
		},
	}

	return &mwaaServerlessStore{
		nextWorkflowID: 2,
		nextRunID:      2,
		nextTaskID:     2,
		workflows: map[string]*mwaaServerlessWorkflow{
			workflowArn: workflow,
		},
		tags: map[string]map[string]string{
			workflowArn: {"env": "coverage", "stackyard": "true"},
		},
	}
}

func (s *mwaaServerlessStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	workflowArn := s.resolveWorkflowArnLocked(payload)
	workflow := s.ensureWorkflowLocked(workflowArn)
	versionID := strings.TrimSpace(mwaaServerlessPayloadString(payload, []string{"WorkflowVersion", "workflowVersion"}, workflow.LatestVersion))
	if versionID == "" {
		versionID = workflow.LatestVersion
	}
	version := s.ensureVersionLocked(workflow, versionID)
	runID := strings.TrimSpace(mwaaServerlessPayloadString(payload, []string{"RunId", "runId"}, "run-000001"))
	if runID == "" {
		runID = "run-000001"
	}
	taskID := strings.TrimSpace(mwaaServerlessPayloadString(payload, []string{"TaskInstanceId", "taskInstanceId"}, "task-000001"))
	if taskID == "" {
		taskID = "task-000001"
	}

	switch action {
	case "CreateWorkflow":
		now := time.Now().UTC().Format(time.RFC3339)
		name := mwaaServerlessPayloadString(payload, []string{"Name", "name"}, "")
		if name == "" {
			name = fmt.Sprintf("stackyard-workflow-%06d", s.nextWorkflowID)
		}
		arn := "arn:aws:mwaa-serverless:us-east-1:123456789012:workflow/" + name
		created := s.ensureWorkflowLocked(arn)
		created.Name = name
		created.Description = mwaaServerlessPayloadString(payload, []string{"Description", "description"}, "stackyard mwaa serverless workflow")
		created.RoleArn = mwaaServerlessPayloadString(payload, []string{"RoleArn", "roleArn"}, "arn:aws:iam::123456789012:role/stackyard-mwaa-serverless-role")
		created.Engine = mwaaServerlessPayloadFloat(payload, []string{"EngineVersion", "engineVersion"}, 2.10)
		created.TriggerMode = mwaaServerlessPayloadString(payload, []string{"TriggerMode", "triggerMode"}, "ON_DEMAND")
		created.Status = "ACTIVE"
		created.UpdatedAt = now
		created.CreatedAt = now
		s.setVersionPayloadLocked(created, created.LatestVersion, payload, now)
		for k, v := range mwaaServerlessPayloadTags(payload) {
			s.ensureTagMapLocked(created.ARN)[k] = v
		}
		return s.workflowMutationResponse(created, created.LatestVersion)

	case "DeleteWorkflow":
		workflow.Status = "DELETED"
		workflow.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{}

	case "GetWorkflow":
		return map[string]any{
			"WorkflowArn":           workflow.ARN,
			"Name":                  workflow.Name,
			"Description":           workflow.Description,
			"RoleArn":               workflow.RoleArn,
			"EngineVersion":         workflow.Engine,
			"TriggerMode":           workflow.TriggerMode,
			"WorkflowStatus":        workflow.Status,
			"CreatedAt":             workflow.CreatedAt,
			"UpdatedAt":             workflow.UpdatedAt,
			"LatestVersion":         workflow.LatestVersion,
			"DefinitionS3Location":  mwaaServerlessCloneAnyMap(version.DefinitionS3Location),
			"LoggingConfiguration":  mwaaServerlessCloneAnyMap(version.LoggingConfiguration),
			"NetworkConfiguration":  mwaaServerlessCloneAnyMap(version.NetworkConfiguration),
			"ScheduleConfiguration": mwaaServerlessCloneAnyMap(version.ScheduleConfiguration),
		}

	case "UpdateWorkflow":
		now := time.Now().UTC().Format(time.RFC3339)
		if name := mwaaServerlessPayloadString(payload, []string{"Name", "name"}, ""); name != "" {
			workflow.Name = name
		}
		if description := mwaaServerlessPayloadString(payload, []string{"Description", "description"}, ""); description != "" {
			workflow.Description = description
		}
		if roleArn := mwaaServerlessPayloadString(payload, []string{"RoleArn", "roleArn"}, ""); roleArn != "" {
			workflow.RoleArn = roleArn
		}
		if triggerMode := mwaaServerlessPayloadString(payload, []string{"TriggerMode", "triggerMode"}, ""); triggerMode != "" {
			workflow.TriggerMode = triggerMode
		}
		workflow.Engine = mwaaServerlessPayloadFloat(payload, []string{"EngineVersion", "engineVersion"}, workflow.Engine)
		workflow.UpdatedAt = now
		workflow.Status = "ACTIVE"

		nextVersion := fmt.Sprintf("%d", len(workflow.Versions)+1)
		workflow.LatestVersion = nextVersion
		workflow.Versions[nextVersion] = &mwaaServerlessWorkflowVersion{
			Version:               nextVersion,
			RevisionID:            fmt.Sprintf("rev-%06d", len(workflow.Versions)),
			CreatedAt:             now,
			UpdatedAt:             now,
			DefinitionS3Location:  map[string]any{},
			LoggingConfiguration:  map[string]any{},
			NetworkConfiguration:  map[string]any{},
			ScheduleConfiguration: map[string]any{},
			Runs:                  map[string]*mwaaServerlessRun{},
		}
		s.setVersionPayloadLocked(workflow, nextVersion, payload, now)
		return s.workflowMutationResponse(workflow, nextVersion)

	case "ListWorkflows":
		items := make([]any, 0, len(s.workflows))
		for _, arn := range s.sortedWorkflowARNsLocked() {
			wf := s.workflows[arn]
			items = append(items, map[string]any{
				"WorkflowArn":    wf.ARN,
				"Name":           wf.Name,
				"Description":    wf.Description,
				"LatestVersion":  wf.LatestVersion,
				"WorkflowStatus": wf.Status,
				"CreatedAt":      wf.CreatedAt,
				"UpdatedAt":      wf.UpdatedAt,
			})
		}
		return map[string]any{"Workflows": items, "NextToken": ""}

	case "ListWorkflowVersions":
		items := make([]any, 0, len(workflow.Versions))
		for _, id := range s.sortedWorkflowVersionIDsLocked(workflow) {
			v := workflow.Versions[id]
			items = append(items, map[string]any{
				"WorkflowVersion": id,
				"RevisionId":      v.RevisionID,
				"IsLatestVersion": id == workflow.LatestVersion,
				"CreatedAt":       v.CreatedAt,
				"UpdatedAt":       v.UpdatedAt,
			})
		}
		return map[string]any{"WorkflowVersions": items, "NextToken": ""}

	case "StartWorkflowRun":
		now := time.Now().UTC().Format(time.RFC3339)
		run := s.createRunLocked(version, now)
		run.Status = "RUNNING"
		return map[string]any{
			"RunId":           run.RunID,
			"StartedAt":       run.StartedAt,
			"WorkflowArn":     workflow.ARN,
			"WorkflowVersion": version.Version,
			"WorkflowStatus":  run.Status,
		}

	case "StopWorkflowRun":
		run := s.ensureRunLocked(version, runID)
		run.Status = "STOPPED"
		run.EndedAt = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"RunId": run.RunID, "WorkflowStatus": run.Status}

	case "GetWorkflowRun":
		run := s.ensureRunLocked(version, runID)
		return map[string]any{
			"WorkflowRunDetail": map[string]any{
				"RunId":           run.RunID,
				"WorkflowArn":     workflow.ARN,
				"WorkflowVersion": version.Version,
				"WorkflowStatus":  run.Status,
				"StartedAt":       run.StartedAt,
				"EndedAt":         run.EndedAt,
			},
		}

	case "ListWorkflowRuns":
		items := make([]any, 0, len(version.Runs))
		for _, id := range s.sortedWorkflowRunIDsLocked(version) {
			run := version.Runs[id]
			items = append(items, map[string]any{
				"RunId":           run.RunID,
				"WorkflowVersion": run.WorkflowVersion,
				"WorkflowStatus":  run.Status,
				"StartedAt":       run.StartedAt,
				"EndedAt":         run.EndedAt,
			})
		}
		return map[string]any{"WorkflowRuns": items, "NextToken": ""}

	case "ListTaskInstances":
		run := s.ensureRunLocked(version, runID)
		items := make([]any, 0, len(run.Tasks))
		for _, id := range s.sortedTaskInstanceIDsLocked(run) {
			task := run.Tasks[id]
			items = append(items, map[string]any{
				"TaskInstanceId": task.TaskInstanceID,
				"TaskName":       task.TaskName,
				"TaskStatus":     task.Status,
				"StartedAt":      task.StartedAt,
				"EndedAt":        task.EndedAt,
			})
		}
		return map[string]any{"TaskInstances": items, "NextToken": ""}

	case "GetTaskInstance":
		run := s.ensureRunLocked(version, runID)
		task := s.ensureTaskInstanceLocked(run, taskID)
		return map[string]any{
			"TaskInstance": map[string]any{
				"TaskInstanceId": task.TaskInstanceID,
				"TaskName":       task.TaskName,
				"TaskStatus":     task.Status,
				"StartedAt":      task.StartedAt,
				"EndedAt":        task.EndedAt,
			},
		}

	case "TagResource":
		arn := mwaaServerlessPayloadString(payload, []string{"ResourceArn", "resourceArn"}, workflow.ARN)
		if arn == "" {
			arn = workflow.ARN
		}
		tagMap := s.ensureTagMapLocked(arn)
		for k, v := range mwaaServerlessPayloadTags(payload) {
			tagMap[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		arn := mwaaServerlessPayloadString(payload, []string{"ResourceArn", "resourceArn"}, workflow.ARN)
		if arn == "" {
			arn = workflow.ARN
		}
		tagMap := s.ensureTagMapLocked(arn)
		for _, key := range mwaaServerlessPayloadTagKeys(payload) {
			delete(tagMap, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		arn := mwaaServerlessPayloadString(payload, []string{"ResourceArn", "resourceArn"}, workflow.ARN)
		if arn == "" {
			arn = workflow.ARN
		}
		return map[string]any{"Tags": mwaaServerlessCloneStringMap(s.ensureTagMapLocked(arn))}
	}

	return map[string]any{}
}

func (s *mwaaServerlessStore) resolveWorkflowArnLocked(payload map[string]any) string {
	arn := strings.TrimSpace(mwaaServerlessPayloadString(payload, []string{"WorkflowArn", "workflowArn", "ResourceArn", "resourceArn"}, ""))
	if arn != "" {
		return arn
	}
	name := strings.TrimSpace(mwaaServerlessPayloadString(payload, []string{"Name", "name"}, ""))
	if name != "" {
		return "arn:aws:mwaa-serverless:us-east-1:123456789012:workflow/" + name
	}
	return "arn:aws:mwaa-serverless:us-east-1:123456789012:workflow/stackyard-workflow"
}

func (s *mwaaServerlessStore) ensureWorkflowLocked(arn string) *mwaaServerlessWorkflow {
	id := strings.TrimSpace(arn)
	if id == "" {
		id = "arn:aws:mwaa-serverless:us-east-1:123456789012:workflow/stackyard-workflow"
	}
	if existing, ok := s.workflows[id]; ok {
		return existing
	}

	now := time.Now().UTC().Format(time.RFC3339)
	name := id
	if idx := strings.LastIndex(id, "/"); idx >= 0 && idx+1 < len(id) {
		name = id[idx+1:]
	}
	created := &mwaaServerlessWorkflow{
		ARN:           id,
		Name:          name,
		Description:   "stackyard mwaa serverless workflow",
		RoleArn:       "arn:aws:iam::123456789012:role/stackyard-mwaa-serverless-role",
		Engine:        2.10,
		TriggerMode:   "ON_DEMAND",
		Status:        "ACTIVE",
		CreatedAt:     now,
		UpdatedAt:     now,
		LatestVersion: "1",
		Versions: map[string]*mwaaServerlessWorkflowVersion{
			"1": {
				Version:               "1",
				RevisionID:            "rev-000001",
				CreatedAt:             now,
				UpdatedAt:             now,
				DefinitionS3Location:  map[string]any{"Bucket": "stackyard-mwaa-serverless", "ObjectKey": "workflows/workflow.yaml", "VersionId": "1"},
				LoggingConfiguration:  map[string]any{"LogGroupName": "/aws/mwaa-serverless/" + name},
				NetworkConfiguration:  map[string]any{},
				ScheduleConfiguration: map[string]any{},
				Runs:                  map[string]*mwaaServerlessRun{},
			},
		},
	}
	s.workflows[id] = created
	s.ensureTagMapLocked(id)
	return created
}

func (s *mwaaServerlessStore) ensureVersionLocked(workflow *mwaaServerlessWorkflow, version string) *mwaaServerlessWorkflowVersion {
	id := strings.TrimSpace(version)
	if id == "" {
		id = workflow.LatestVersion
	}
	if existing, ok := workflow.Versions[id]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &mwaaServerlessWorkflowVersion{
		Version:               id,
		RevisionID:            fmt.Sprintf("rev-%06d", len(workflow.Versions)+1),
		CreatedAt:             now,
		UpdatedAt:             now,
		DefinitionS3Location:  map[string]any{},
		LoggingConfiguration:  map[string]any{},
		NetworkConfiguration:  map[string]any{},
		ScheduleConfiguration: map[string]any{},
		Runs:                  map[string]*mwaaServerlessRun{},
	}
	workflow.Versions[id] = created
	if workflow.LatestVersion == "" {
		workflow.LatestVersion = id
	}
	return created
}

func (s *mwaaServerlessStore) createRunLocked(version *mwaaServerlessWorkflowVersion, now string) *mwaaServerlessRun {
	runID := fmt.Sprintf("run-%06d", s.nextRunID)
	s.nextRunID++
	run := &mwaaServerlessRun{
		RunID:           runID,
		WorkflowVersion: version.Version,
		Status:          "QUEUED",
		StartedAt:       now,
		Tasks:           map[string]*mwaaServerlessTaskInstance{},
	}
	task := s.ensureTaskInstanceLocked(run, "")
	task.Status = "RUNNING"
	task.StartedAt = now
	version.Runs[runID] = run
	return run
}

func (s *mwaaServerlessStore) ensureRunLocked(version *mwaaServerlessWorkflowVersion, runID string) *mwaaServerlessRun {
	id := strings.TrimSpace(runID)
	if id == "" {
		id = "run-000001"
	}
	if existing, ok := version.Runs[id]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &mwaaServerlessRun{
		RunID:           id,
		WorkflowVersion: version.Version,
		Status:          "SUCCEEDED",
		StartedAt:       now,
		EndedAt:         now,
		Tasks:           map[string]*mwaaServerlessTaskInstance{},
	}
	created.Tasks["task-000001"] = &mwaaServerlessTaskInstance{
		TaskInstanceID: "task-000001",
		TaskName:       "seed-task",
		Status:         "SUCCEEDED",
		StartedAt:      now,
		EndedAt:        now,
	}
	version.Runs[id] = created
	return created
}

func (s *mwaaServerlessStore) ensureTaskInstanceLocked(run *mwaaServerlessRun, taskID string) *mwaaServerlessTaskInstance {
	id := strings.TrimSpace(taskID)
	if id == "" {
		id = fmt.Sprintf("task-%06d", s.nextTaskID)
		s.nextTaskID++
	}
	if existing, ok := run.Tasks[id]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &mwaaServerlessTaskInstance{
		TaskInstanceID: id,
		TaskName:       "task-" + id,
		Status:         "SUCCEEDED",
		StartedAt:      now,
		EndedAt:        now,
	}
	run.Tasks[id] = created
	return created
}

func (s *mwaaServerlessStore) setVersionPayloadLocked(
	workflow *mwaaServerlessWorkflow,
	versionID string,
	payload map[string]any,
	now string,
) {
	version := s.ensureVersionLocked(workflow, versionID)
	if def := mwaaServerlessPayloadMap(payload, []string{"DefinitionS3Location", "definitionS3Location"}); len(def) > 0 {
		version.DefinitionS3Location = def
	}
	if logging := mwaaServerlessPayloadMap(payload, []string{"LoggingConfiguration", "loggingConfiguration"}); len(logging) > 0 {
		version.LoggingConfiguration = logging
	}
	if network := mwaaServerlessPayloadMap(payload, []string{"NetworkConfiguration", "networkConfiguration"}); len(network) > 0 {
		version.NetworkConfiguration = network
	}
	if schedule := mwaaServerlessPayloadMap(payload, []string{"ScheduleConfiguration", "scheduleConfiguration"}); len(schedule) > 0 {
		version.ScheduleConfiguration = schedule
	}
	version.UpdatedAt = now
}

func (s *mwaaServerlessStore) workflowMutationResponse(workflow *mwaaServerlessWorkflow, versionID string) map[string]any {
	version := s.ensureVersionLocked(workflow, versionID)
	return map[string]any{
		"CreatedAt":       workflow.CreatedAt,
		"IsLatestVersion": versionID == workflow.LatestVersion,
		"RevisionId":      version.RevisionID,
		"Warnings":        []any{},
		"WorkflowArn":     workflow.ARN,
		"WorkflowStatus":  workflow.Status,
		"WorkflowVersion": versionID,
	}
}

func (s *mwaaServerlessStore) ensureTagMapLocked(arn string) map[string]string {
	id := strings.TrimSpace(arn)
	if id == "" {
		id = "arn:aws:mwaa-serverless:us-east-1:123456789012:workflow/stackyard-workflow"
	}
	if existing, ok := s.tags[id]; ok {
		return existing
	}
	created := map[string]string{}
	s.tags[id] = created
	return created
}

func (s *mwaaServerlessStore) sortedWorkflowARNsLocked() []string {
	out := make([]string, 0, len(s.workflows))
	for arn := range s.workflows {
		out = append(out, arn)
	}
	sort.Strings(out)
	return out
}

func (s *mwaaServerlessStore) sortedWorkflowVersionIDsLocked(workflow *mwaaServerlessWorkflow) []string {
	out := make([]string, 0, len(workflow.Versions))
	for version := range workflow.Versions {
		out = append(out, version)
	}
	sort.Strings(out)
	return out
}

func (s *mwaaServerlessStore) sortedWorkflowRunIDsLocked(version *mwaaServerlessWorkflowVersion) []string {
	out := make([]string, 0, len(version.Runs))
	for runID := range version.Runs {
		out = append(out, runID)
	}
	sort.Strings(out)
	return out
}

func (s *mwaaServerlessStore) sortedTaskInstanceIDsLocked(run *mwaaServerlessRun) []string {
	out := make([]string, 0, len(run.Tasks))
	for taskID := range run.Tasks {
		out = append(out, taskID)
	}
	sort.Strings(out)
	return out
}

func mwaaServerlessPayloadString(payload map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		if raw := mwaaServerlessPayloadValue(payload, key); raw != nil {
			if value, ok := raw.(string); ok {
				trimmed := strings.TrimSpace(value)
				if trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return fallback
}

func mwaaServerlessPayloadFloat(payload map[string]any, keys []string, fallback float64) float64 {
	for _, key := range keys {
		raw := mwaaServerlessPayloadValue(payload, key)
		switch v := raw.(type) {
		case json.Number:
			if f, err := v.Float64(); err == nil {
				return f
			}
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case string:
			if n, err := json.Number(strings.TrimSpace(v)).Float64(); err == nil {
				return n
			}
		}
	}
	return fallback
}

func mwaaServerlessPayloadMap(payload map[string]any, keys []string) map[string]any {
	for _, key := range keys {
		if raw := mwaaServerlessPayloadValue(payload, key); raw != nil {
			if value, ok := raw.(map[string]any); ok {
				return mwaaServerlessCloneAnyMap(value)
			}
		}
	}
	return map[string]any{}
}

func mwaaServerlessPayloadTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	raw := mwaaServerlessPayloadValue(payload, "Tags")
	if raw == nil {
		raw = mwaaServerlessPayloadValue(payload, "tags")
	}
	if value, ok := raw.(map[string]any); ok {
		for k, v := range value {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprint(v))
		}
	}
	if value, ok := raw.(map[string]string); ok {
		for k, v := range value {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(v)
		}
	}
	return out
}

func mwaaServerlessPayloadTagKeys(payload map[string]any) []string {
	raw := mwaaServerlessPayloadValue(payload, "TagKeys")
	if raw == nil {
		raw = mwaaServerlessPayloadValue(payload, "tagKeys")
	}
	out := []string{}
	switch v := raw.(type) {
	case []any:
		for _, entry := range v {
			key := strings.TrimSpace(fmt.Sprint(entry))
			if key != "" {
				out = append(out, key)
			}
		}
	case []string:
		for _, entry := range v {
			key := strings.TrimSpace(entry)
			if key != "" {
				out = append(out, key)
			}
		}
	case string:
		for _, entry := range strings.Split(v, ",") {
			key := strings.TrimSpace(entry)
			if key != "" {
				out = append(out, key)
			}
		}
	}
	return out
}

func mwaaServerlessPayloadValue(payload map[string]any, want string) any {
	if payload == nil {
		return nil
	}
	if value, ok := payload[want]; ok {
		return value
	}
	for key, value := range payload {
		if strings.EqualFold(strings.TrimSpace(key), strings.TrimSpace(want)) {
			return value
		}
	}
	return nil
}

func mwaaServerlessCloneAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mwaaServerlessCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
