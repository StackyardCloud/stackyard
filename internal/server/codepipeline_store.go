package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type codePipelineStore struct {
	mu                sync.Mutex
	nextID            int64
	pipelines         map[string]*codePipelinePipeline
	executions        map[string][]*codePipelineExecution
	customActionTypes map[string]map[string]any
	webhooks          map[string]map[string]any
	tags              map[string]map[string]string
	jobs              map[string]*codePipelineJob
	thirdPartyJobs    map[string]*codePipelineJob
}

type codePipelinePipeline struct {
	Name      string
	Version   int
	Structure map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

type codePipelineExecution struct {
	ID         string
	Pipeline   string
	Status     string
	StartTime  time.Time
	LastUpdate time.Time
}

type codePipelineJob struct {
	ID         string
	Nonce      string
	Status     string
	CreatedAt  time.Time
	ActionType map[string]any
}

func newCodePipelineStore() *codePipelineStore {
	now := time.Now().UTC()
	defaultPipeline := &codePipelinePipeline{
		Name:      "stackyard-default-pipeline",
		Version:   1,
		Structure: codePipelineDefaultPipeline("stackyard-default-pipeline"),
		CreatedAt: now,
		UpdatedAt: now,
	}

	defaultActionType := map[string]any{
		"id": map[string]any{
			"category": "Build",
			"owner":    "AWS",
			"provider": "CodeBuild",
			"version":  "1",
		},
		"settings": map[string]any{
			"entityUrlTemplate":    "",
			"executionUrlTemplate": "",
			"revisionUrlTemplate":  "",
		},
		"inputArtifactDetails":  map[string]any{"minimumCount": 0, "maximumCount": 5},
		"outputArtifactDetails": map[string]any{"minimumCount": 0, "maximumCount": 5},
	}

	job := &codePipelineJob{
		ID:        "job-stackyard-000001",
		Nonce:     "nonce-stackyard-000001",
		Status:    "Created",
		CreatedAt: now,
		ActionType: map[string]any{
			"category": "Build",
			"owner":    "Custom",
			"provider": "StackyardWorker",
			"version":  "1",
		},
	}

	thirdPartyJob := &codePipelineJob{
		ID:        "thirdparty-job-stackyard-000001",
		Nonce:     "nonce-thirdparty-000001",
		Status:    "Created",
		CreatedAt: now,
		ActionType: map[string]any{
			"category": "Source",
			"owner":    "ThirdParty",
			"provider": "StackyardPartner",
			"version":  "1",
		},
	}

	return &codePipelineStore{
		nextID: 1,
		pipelines: map[string]*codePipelinePipeline{
			defaultPipeline.Name: defaultPipeline,
		},
		executions: map[string][]*codePipelineExecution{
			defaultPipeline.Name: {},
		},
		customActionTypes: map[string]map[string]any{
			"AWS|CodeBuild|Build|1": codePipelineCloneMapAny(defaultActionType),
		},
		webhooks: map[string]map[string]any{},
		tags:     map[string]map[string]string{},
		jobs: map[string]*codePipelineJob{
			job.ID: job,
		},
		thirdPartyJobs: map[string]*codePipelineJob{
			thirdPartyJob.ID: thirdPartyJob,
		},
	}
}

func (s *codePipelineStore) CreatePipeline(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	pipelinePayload, _ := payload["pipeline"].(map[string]any)
	name := strings.TrimSpace(stringField(pipelinePayload, "name"))
	if name == "" {
		name = fmt.Sprintf("stackyard-pipeline-%06d", s.next())
	}
	if pipelinePayload == nil {
		pipelinePayload = codePipelineDefaultPipeline(name)
	} else if _, ok := pipelinePayload["name"]; !ok {
		pipelinePayload = codePipelineCloneMapAny(pipelinePayload)
		pipelinePayload["name"] = name
	}

	now := time.Now().UTC()
	record, exists := s.pipelines[name]
	if exists {
		record.Version++
		record.Structure = codePipelineCloneMapAny(pipelinePayload)
		record.UpdatedAt = now
	} else {
		record = &codePipelinePipeline{
			Name:      name,
			Version:   1,
			Structure: codePipelineCloneMapAny(pipelinePayload),
			CreatedAt: now,
			UpdatedAt: now,
		}
		s.pipelines[name] = record
		s.executions[name] = []*codePipelineExecution{}
	}

	return map[string]any{
		"pipeline": codePipelineCloneMapAny(record.Structure),
		"tags":     []any{},
	}
}

func (s *codePipelineStore) UpdatePipeline(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	pipelinePayload, _ := payload["pipeline"].(map[string]any)
	name := strings.TrimSpace(stringField(pipelinePayload, "name"))
	if name == "" {
		name = "stackyard-default-pipeline"
	}
	if pipelinePayload == nil {
		pipelinePayload = codePipelineDefaultPipeline(name)
	} else if _, ok := pipelinePayload["name"]; !ok {
		pipelinePayload = codePipelineCloneMapAny(pipelinePayload)
		pipelinePayload["name"] = name
	}

	now := time.Now().UTC()
	record, exists := s.pipelines[name]
	if !exists {
		record = &codePipelinePipeline{Name: name, Version: 1, CreatedAt: now}
		s.pipelines[name] = record
		s.executions[name] = []*codePipelineExecution{}
	}
	record.Version++
	record.Structure = codePipelineCloneMapAny(pipelinePayload)
	record.UpdatedAt = now
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}

	return map[string]any{
		"pipeline": codePipelineCloneMapAny(record.Structure),
		"tags":     []any{},
	}
}

func (s *codePipelineStore) DeletePipeline(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(stringField(payload, "name"))
	if name == "" {
		name = strings.TrimSpace(stringField(payload, "pipelineName"))
	}
	if name != "" {
		delete(s.pipelines, name)
		delete(s.executions, name)
	}
	return map[string]any{}
}

func (s *codePipelineStore) GetPipeline(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.ensurePipelineLocked(payload)
	return map[string]any{
		"pipeline": codePipelineCloneMapAny(record.Structure),
		"metadata": map[string]any{
			"pipelineArn":       codePipelinePipelineARN(record.Name),
			"created":           record.CreatedAt,
			"updated":           record.UpdatedAt,
			"pollingDisabledAt": nil,
		},
	}
}

func (s *codePipelineStore) GetPipelineState(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.ensurePipelineLocked(payload)
	execs := s.executions[record.Name]
	latestStatus := "Succeeded"
	if len(execs) > 0 {
		latestStatus = execs[len(execs)-1].Status
	}

	stages := []map[string]any{
		{
			"stageName": "Source",
			"latestExecution": map[string]any{
				"status": latestStatus,
			},
			"inboundTransitionState": map[string]any{
				"enabled": true,
			},
		},
		{
			"stageName": "Build",
			"latestExecution": map[string]any{
				"status": latestStatus,
			},
			"inboundTransitionState": map[string]any{
				"enabled": true,
			},
		},
	}

	return map[string]any{
		"pipelineName":    record.Name,
		"pipelineVersion": record.Version,
		"stageStates":     stages,
	}
}

func (s *codePipelineStore) ListPipelines() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	names := make([]string, 0, len(s.pipelines))
	for name := range s.pipelines {
		names = append(names, name)
	}
	sort.Strings(names)

	summaries := make([]map[string]any, 0, len(names))
	for _, name := range names {
		record := s.pipelines[name]
		summaries = append(summaries, map[string]any{
			"name":    record.Name,
			"version": record.Version,
			"created": record.CreatedAt,
			"updated": record.UpdatedAt,
		})
	}

	return map[string]any{"pipelines": summaries}
}

func (s *codePipelineStore) StartPipelineExecution(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.ensurePipelineLocked(payload)
	now := time.Now().UTC()
	exec := &codePipelineExecution{
		ID:         fmt.Sprintf("exec-%012d", s.next()),
		Pipeline:   record.Name,
		Status:     "InProgress",
		StartTime:  now,
		LastUpdate: now,
	}
	s.executions[record.Name] = append(s.executions[record.Name], exec)

	return map[string]any{"pipelineExecutionId": exec.ID}
}

func (s *codePipelineStore) ListPipelineExecutions(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.ensurePipelineLocked(payload)
	execs := s.executions[record.Name]
	if len(execs) == 0 {
		now := time.Now().UTC()
		execs = append(execs, &codePipelineExecution{
			ID:         fmt.Sprintf("exec-%012d", s.next()),
			Pipeline:   record.Name,
			Status:     "Succeeded",
			StartTime:  now,
			LastUpdate: now,
		})
		s.executions[record.Name] = execs
	}

	summaries := make([]map[string]any, 0, len(execs))
	for i := len(execs) - 1; i >= 0; i-- {
		exec := execs[i]
		summaries = append(summaries, map[string]any{
			"pipelineExecutionId": exec.ID,
			"status":              exec.Status,
			"startTime":           exec.StartTime,
			"lastUpdateTime":      exec.LastUpdate,
		})
	}

	return map[string]any{"pipelineExecutionSummaries": summaries}
}

func (s *codePipelineStore) GetPipelineExecution(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.ensurePipelineLocked(payload)
	executionID := strings.TrimSpace(stringField(payload, "pipelineExecutionId"))
	exec := s.findExecutionLocked(record.Name, executionID)
	if exec == nil {
		now := time.Now().UTC()
		exec = &codePipelineExecution{
			ID:         fmt.Sprintf("exec-%012d", s.next()),
			Pipeline:   record.Name,
			Status:     "Succeeded",
			StartTime:  now,
			LastUpdate: now,
		}
		s.executions[record.Name] = append(s.executions[record.Name], exec)
	}

	return map[string]any{
		"pipelineExecution": map[string]any{
			"pipelineName":        record.Name,
			"pipelineVersion":     record.Version,
			"pipelineExecutionId": exec.ID,
			"status":              exec.Status,
		},
	}
}

func (s *codePipelineStore) StopPipelineExecution(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.ensurePipelineLocked(payload)
	executionID := strings.TrimSpace(stringField(payload, "pipelineExecutionId"))
	exec := s.findExecutionLocked(record.Name, executionID)
	if exec == nil {
		now := time.Now().UTC()
		exec = &codePipelineExecution{
			ID:         fmt.Sprintf("exec-%012d", s.next()),
			Pipeline:   record.Name,
			Status:     "Stopped",
			StartTime:  now,
			LastUpdate: now,
		}
		s.executions[record.Name] = append(s.executions[record.Name], exec)
	} else {
		exec.Status = "Stopped"
		exec.LastUpdate = time.Now().UTC()
	}
	return map[string]any{"pipelineExecutionId": exec.ID}
}

func (s *codePipelineStore) RetryStageExecution(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.ensurePipelineLocked(payload)
	now := time.Now().UTC()
	exec := &codePipelineExecution{
		ID:         fmt.Sprintf("exec-%012d", s.next()),
		Pipeline:   record.Name,
		Status:     "InProgress",
		StartTime:  now,
		LastUpdate: now,
	}
	s.executions[record.Name] = append(s.executions[record.Name], exec)
	return map[string]any{"pipelineExecutionId": exec.ID}
}

func (s *codePipelineStore) RollbackStage(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.ensurePipelineLocked(payload)
	now := time.Now().UTC()
	exec := &codePipelineExecution{
		ID:         fmt.Sprintf("exec-%012d", s.next()),
		Pipeline:   record.Name,
		Status:     "InProgress",
		StartTime:  now,
		LastUpdate: now,
	}
	s.executions[record.Name] = append(s.executions[record.Name], exec)
	return map[string]any{"pipelineExecutionId": exec.ID}
}

func (s *codePipelineStore) OverrideStageCondition() map[string]any {
	return map[string]any{}
}

func (s *codePipelineStore) EnableStageTransition() map[string]any {
	return map[string]any{}
}

func (s *codePipelineStore) DisableStageTransition() map[string]any {
	return map[string]any{}
}

func (s *codePipelineStore) ListActionExecutions(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.ensurePipelineLocked(payload)
	execs := s.executions[record.Name]
	if len(execs) == 0 {
		now := time.Now().UTC()
		execs = append(execs, &codePipelineExecution{
			ID:         fmt.Sprintf("exec-%012d", s.next()),
			Pipeline:   record.Name,
			Status:     "Succeeded",
			StartTime:  now,
			LastUpdate: now,
		})
		s.executions[record.Name] = execs
	}
	latest := execs[len(execs)-1]

	return map[string]any{
		"actionExecutionDetails": []map[string]any{
			{
				"pipelineExecutionId": latest.ID,
				"pipelineVersion":     record.Version,
				"stageName":           "Build",
				"actionName":          "Build",
				"status":              latest.Status,
				"startTime":           latest.StartTime,
				"lastUpdateTime":      latest.LastUpdate,
			},
		},
	}
}

func (s *codePipelineStore) ListDeployActionExecutionTargets(payload map[string]any) map[string]any {
	_ = payload
	return map[string]any{
		"targets": []map[string]any{
			{
				"targetId":      "stackyard-target-1",
				"targetType":    "CloudFormationStack",
				"status":        "Succeeded",
				"lastUpdatedAt": time.Now().UTC(),
			},
		},
	}
}

func (s *codePipelineStore) CreateCustomActionType(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	actionType, _ := payload["actionType"].(map[string]any)
	key := customActionTypeKey(actionType)
	if key == "" {
		key = fmt.Sprintf("Custom|Stackyard|Build|%d", s.next())
		actionType = map[string]any{
			"id": map[string]any{
				"owner":    "Custom",
				"provider": "Stackyard",
				"category": "Build",
				"version":  fmt.Sprintf("%d", s.next()),
			},
		}
	}
	s.customActionTypes[key] = codePipelineCloneMapAny(actionType)
	return map[string]any{}
}

func (s *codePipelineStore) DeleteCustomActionType(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	actionType, _ := payload["actionType"].(map[string]any)
	key := customActionTypeKey(actionType)
	if key != "" {
		delete(s.customActionTypes, key)
	}
	return map[string]any{}
}

func (s *codePipelineStore) UpdateActionType(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	actionType, _ := payload["actionType"].(map[string]any)
	if actionType == nil {
		actionType = map[string]any{
			"id": map[string]any{
				"owner":    "Custom",
				"provider": "Stackyard",
				"category": "Build",
				"version":  "1",
			},
		}
	}
	key := customActionTypeKey(actionType)
	if key == "" {
		key = fmt.Sprintf("Custom|Stackyard|Build|%d", s.next())
	}
	s.customActionTypes[key] = codePipelineCloneMapAny(actionType)
	return map[string]any{}
}

func (s *codePipelineStore) GetActionType(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	identifier, _ := payload["actionType"].(map[string]any)
	key := customActionTypeKey(identifier)
	if key != "" {
		if found, ok := s.customActionTypes[key]; ok {
			return map[string]any{"actionType": codePipelineCloneMapAny(found)}
		}
	}
	for _, found := range s.customActionTypes {
		return map[string]any{"actionType": codePipelineCloneMapAny(found)}
	}
	return map[string]any{"actionType": map[string]any{}}
}

func (s *codePipelineStore) ListActionTypes() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]map[string]any, 0, len(s.customActionTypes))
	for _, actionType := range s.customActionTypes {
		items = append(items, codePipelineCloneMapAny(actionType))
	}
	sort.Slice(items, func(i, j int) bool {
		return customActionTypeKey(items[i]) < customActionTypeKey(items[j])
	})
	return map[string]any{"actionTypes": items}
}

func (s *codePipelineStore) ListRuleTypes() map[string]any {
	return map[string]any{
		"ruleTypes": []map[string]any{
			{
				"id": map[string]any{
					"category": "Rule",
					"owner":    "AWS",
					"provider": "Lambda",
					"version":  "1",
				},
				"settings": map[string]any{
					"entityUrlTemplate":    "",
					"executionUrlTemplate": "",
					"revisionUrlTemplate":  "",
				},
			},
		},
	}
}

func (s *codePipelineStore) ListRuleExecutions(payload map[string]any) map[string]any {
	_ = payload
	return map[string]any{"ruleExecutionDetails": []any{}}
}

func (s *codePipelineStore) PutWebhook(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	webhook, _ := payload["webhook"].(map[string]any)
	name := strings.TrimSpace(stringField(webhook, "name"))
	if name == "" {
		name = fmt.Sprintf("stackyard-webhook-%06d", s.next())
	}
	if webhook == nil {
		webhook = map[string]any{}
	}
	copyWebhook := codePipelineCloneMapAny(webhook)
	copyWebhook["name"] = name
	if _, ok := copyWebhook["url"]; !ok {
		copyWebhook["url"] = fmt.Sprintf("https://example.com/%s", name)
	}
	if _, ok := copyWebhook["registeredWithThirdParty"]; !ok {
		copyWebhook["registeredWithThirdParty"] = false
	}
	s.webhooks[name] = copyWebhook
	return map[string]any{"webhook": codePipelineCloneMapAny(copyWebhook)}
}

func (s *codePipelineStore) ListWebhooks() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	names := make([]string, 0, len(s.webhooks))
	for name := range s.webhooks {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]map[string]any, 0, len(names))
	for _, name := range names {
		items = append(items, map[string]any{"definition": codePipelineCloneMapAny(s.webhooks[name])})
	}
	return map[string]any{"webhooks": items}
}

func (s *codePipelineStore) DeleteWebhook(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(stringField(payload, "name"))
	if name != "" {
		delete(s.webhooks, name)
	}
	return map[string]any{}
}

func (s *codePipelineStore) RegisterWebhookWithThirdParty(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(stringField(payload, "webhookName"))
	if name == "" {
		name = strings.TrimSpace(stringField(payload, "name"))
	}
	if name == "" {
		for existing := range s.webhooks {
			name = existing
			break
		}
	}
	if name != "" {
		if record, ok := s.webhooks[name]; ok {
			record["registeredWithThirdParty"] = true
		}
	}
	return map[string]any{}
}

func (s *codePipelineStore) DeregisterWebhookWithThirdParty(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(stringField(payload, "webhookName"))
	if name == "" {
		name = strings.TrimSpace(stringField(payload, "name"))
	}
	if name == "" {
		for existing := range s.webhooks {
			name = existing
			break
		}
	}
	if name != "" {
		if record, ok := s.webhooks[name]; ok {
			record["registeredWithThirdParty"] = false
		}
	}
	return map[string]any{}
}

func (s *codePipelineStore) TagResource(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceArn := strings.TrimSpace(stringField(payload, "resourceArn"))
	if resourceArn == "" {
		resourceArn = codePipelinePipelineARN("stackyard-default-pipeline")
	}
	if _, ok := s.tags[resourceArn]; !ok {
		s.tags[resourceArn] = map[string]string{}
	}

	rawTags, _ := payload["tags"].([]any)
	for _, raw := range rawTags {
		tag, _ := raw.(map[string]any)
		key := strings.TrimSpace(stringField(tag, "key"))
		if key == "" {
			continue
		}
		s.tags[resourceArn][key] = strings.TrimSpace(stringField(tag, "value"))
	}
	return map[string]any{}
}

func (s *codePipelineStore) UntagResource(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceArn := strings.TrimSpace(stringField(payload, "resourceArn"))
	if resourceArn == "" {
		resourceArn = codePipelinePipelineARN("stackyard-default-pipeline")
	}
	tags := s.tags[resourceArn]
	rawKeys, _ := payload["tagKeys"].([]any)
	for _, raw := range rawKeys {
		if key, ok := raw.(string); ok {
			delete(tags, strings.TrimSpace(key))
		}
	}
	return map[string]any{}
}

func (s *codePipelineStore) ListTagsForResource(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceArn := strings.TrimSpace(stringField(payload, "resourceArn"))
	if resourceArn == "" {
		resourceArn = codePipelinePipelineARN("stackyard-default-pipeline")
	}
	rawTags := s.tags[resourceArn]
	keys := make([]string, 0, len(rawTags))
	for key := range rawTags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	tags := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		tags = append(tags, map[string]any{"key": key, "value": rawTags[key]})
	}
	return map[string]any{"tags": tags}
}

func (s *codePipelineStore) PollForJobs() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := s.ensureJobLocked(false)
	return map[string]any{
		"jobs": []map[string]any{
			{
				"id":    job.ID,
				"nonce": job.Nonce,
				"data": map[string]any{
					"actionTypeId": codePipelineCloneMapAny(job.ActionType),
					"pipelineContext": map[string]any{
						"pipelineName": "stackyard-default-pipeline",
					},
				},
			},
		},
	}
}

func (s *codePipelineStore) AcknowledgeJob(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID := strings.TrimSpace(stringField(payload, "jobId"))
	job := s.ensureJobLocked(false)
	if jobID != "" {
		if found, ok := s.jobs[jobID]; ok {
			job = found
		}
	}
	job.Status = "InProgress"
	return map[string]any{"status": "InProgress"}
}

func (s *codePipelineStore) GetJobDetails(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID := strings.TrimSpace(stringField(payload, "jobId"))
	job := s.ensureJobLocked(false)
	if jobID != "" {
		if found, ok := s.jobs[jobID]; ok {
			job = found
		}
	}
	return map[string]any{
		"jobDetails": map[string]any{
			"id":        job.ID,
			"accountId": "123456789012",
			"status":    job.Status,
			"data": map[string]any{
				"actionTypeId":    codePipelineCloneMapAny(job.ActionType),
				"pipelineContext": map[string]any{"pipelineName": "stackyard-default-pipeline"},
			},
		},
	}
}

func (s *codePipelineStore) PutJobSuccessResult(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID := strings.TrimSpace(stringField(payload, "jobId"))
	if jobID == "" {
		jobID = s.ensureJobLocked(false).ID
	}
	if job, ok := s.jobs[jobID]; ok {
		job.Status = "Succeeded"
	}
	return map[string]any{}
}

func (s *codePipelineStore) PutJobFailureResult(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID := strings.TrimSpace(stringField(payload, "jobId"))
	if jobID == "" {
		jobID = s.ensureJobLocked(false).ID
	}
	if job, ok := s.jobs[jobID]; ok {
		job.Status = "Failed"
	}
	return map[string]any{}
}

func (s *codePipelineStore) PutActionRevision(payload map[string]any) map[string]any {
	pipelineName := strings.TrimSpace(stringField(payload, "pipelineName"))
	if pipelineName == "" {
		pipelineName = "stackyard-default-pipeline"
	}
	return map[string]any{
		"newRevision":         true,
		"pipelineExecutionId": fmt.Sprintf("exec-%s", pipelineName),
	}
}

func (s *codePipelineStore) PutApprovalResult() map[string]any {
	return map[string]any{"approvedAt": time.Now().UTC()}
}

func (s *codePipelineStore) PollForThirdPartyJobs() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := s.ensureJobLocked(true)
	return map[string]any{
		"jobs": []map[string]any{{
			"clientId":     "stackyard-thirdparty-client",
			"jobId":        job.ID,
			"nonce":        job.Nonce,
			"actionTypeId": codePipelineCloneMapAny(job.ActionType),
		}},
	}
}

func (s *codePipelineStore) AcknowledgeThirdPartyJob(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID := strings.TrimSpace(stringField(payload, "jobId"))
	job := s.ensureJobLocked(true)
	if jobID != "" {
		if found, ok := s.thirdPartyJobs[jobID]; ok {
			job = found
		}
	}
	job.Status = "InProgress"
	return map[string]any{"status": "InProgress"}
}

func (s *codePipelineStore) GetThirdPartyJobDetails(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID := strings.TrimSpace(stringField(payload, "jobId"))
	job := s.ensureJobLocked(true)
	if jobID != "" {
		if found, ok := s.thirdPartyJobs[jobID]; ok {
			job = found
		}
	}
	return map[string]any{
		"jobDetails": map[string]any{
			"id":     job.ID,
			"status": job.Status,
			"data": map[string]any{
				"actionTypeId": codePipelineCloneMapAny(job.ActionType),
			},
		},
	}
}

func (s *codePipelineStore) PutThirdPartyJobSuccessResult(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID := strings.TrimSpace(stringField(payload, "jobId"))
	if jobID == "" {
		jobID = s.ensureJobLocked(true).ID
	}
	if job, ok := s.thirdPartyJobs[jobID]; ok {
		job.Status = "Succeeded"
	}
	return map[string]any{}
}

func (s *codePipelineStore) PutThirdPartyJobFailureResult(payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID := strings.TrimSpace(stringField(payload, "jobId"))
	if jobID == "" {
		jobID = s.ensureJobLocked(true).ID
	}
	if job, ok := s.thirdPartyJobs[jobID]; ok {
		job.Status = "Failed"
	}
	return map[string]any{}
}

func (s *codePipelineStore) ensurePipelineLocked(payload map[string]any) *codePipelinePipeline {
	name := strings.TrimSpace(stringField(payload, "name"))
	if name == "" {
		name = strings.TrimSpace(stringField(payload, "pipelineName"))
	}
	if name == "" {
		name = "stackyard-default-pipeline"
	}
	if existing, ok := s.pipelines[name]; ok {
		return existing
	}
	now := time.Now().UTC()
	record := &codePipelinePipeline{
		Name:      name,
		Version:   1,
		Structure: codePipelineDefaultPipeline(name),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.pipelines[name] = record
	s.executions[name] = []*codePipelineExecution{}
	return record
}

func (s *codePipelineStore) findExecutionLocked(pipelineName, executionID string) *codePipelineExecution {
	execs := s.executions[pipelineName]
	if executionID == "" {
		if len(execs) > 0 {
			return execs[len(execs)-1]
		}
		return nil
	}
	for _, execution := range execs {
		if execution.ID == executionID {
			return execution
		}
	}
	return nil
}

func (s *codePipelineStore) ensureJobLocked(thirdParty bool) *codePipelineJob {
	if thirdParty {
		for _, job := range s.thirdPartyJobs {
			return job
		}
		now := time.Now().UTC()
		job := &codePipelineJob{
			ID:        fmt.Sprintf("thirdparty-job-%012d", s.next()),
			Nonce:     fmt.Sprintf("nonce-thirdparty-%012d", s.next()),
			Status:    "Created",
			CreatedAt: now,
			ActionType: map[string]any{
				"category": "Source",
				"owner":    "ThirdParty",
				"provider": "StackyardPartner",
				"version":  "1",
			},
		}
		s.thirdPartyJobs[job.ID] = job
		return job
	}

	for _, job := range s.jobs {
		return job
	}
	now := time.Now().UTC()
	job := &codePipelineJob{
		ID:        fmt.Sprintf("job-%012d", s.next()),
		Nonce:     fmt.Sprintf("nonce-%012d", s.next()),
		Status:    "Created",
		CreatedAt: now,
		ActionType: map[string]any{
			"category": "Build",
			"owner":    "Custom",
			"provider": "StackyardWorker",
			"version":  "1",
		},
	}
	s.jobs[job.ID] = job
	return job
}

func (s *codePipelineStore) next() int64 {
	s.nextID++
	return s.nextID
}

func customActionTypeKey(actionType map[string]any) string {
	identifier, _ := actionType["id"].(map[string]any)
	owner := strings.TrimSpace(stringField(identifier, "owner"))
	provider := strings.TrimSpace(stringField(identifier, "provider"))
	category := strings.TrimSpace(stringField(identifier, "category"))
	version := strings.TrimSpace(stringField(identifier, "version"))
	if owner == "" && provider == "" && category == "" && version == "" {
		return ""
	}
	return owner + "|" + provider + "|" + category + "|" + version
}

func codePipelinePipelineARN(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "stackyard-default-pipeline"
	}
	return "arn:aws:codepipeline:us-east-1:123456789012:" + trimmed
}

func codePipelineDefaultPipeline(name string) map[string]any {
	return map[string]any{
		"name":    name,
		"version": 1,
		"roleArn": "arn:aws:iam::123456789012:role/stackyard-codepipeline",
		"artifactStore": map[string]any{
			"type":     "S3",
			"location": "stackyard-codepipeline-artifacts",
		},
		"stages": []map[string]any{
			{
				"name": "Source",
				"actions": []map[string]any{
					{
						"name":     "Source",
						"runOrder": 1,
						"actionTypeId": map[string]any{
							"category": "Source",
							"owner":    "AWS",
							"provider": "S3",
							"version":  "1",
						},
						"configuration": map[string]any{
							"S3Bucket":    "stackyard-source",
							"S3ObjectKey": "source.zip",
						},
						"outputArtifacts": []map[string]any{{"name": "SourceArtifact"}},
					},
				},
			},
			{
				"name": "Build",
				"actions": []map[string]any{
					{
						"name":     "Build",
						"runOrder": 1,
						"actionTypeId": map[string]any{
							"category": "Build",
							"owner":    "AWS",
							"provider": "CodeBuild",
							"version":  "1",
						},
						"inputArtifacts":  []map[string]any{{"name": "SourceArtifact"}},
						"outputArtifacts": []map[string]any{{"name": "BuildArtifact"}},
					},
				},
			},
		},
	}
}

func codePipelineCloneMapAny(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = codePipelineCloneMapAny(typed)
		case []any:
			out[key] = codePipelineCloneSliceAny(typed)
		case []map[string]any:
			items := make([]map[string]any, 0, len(typed))
			for _, item := range typed {
				items = append(items, codePipelineCloneMapAny(item))
			}
			out[key] = items
		default:
			out[key] = value
		}
	}
	return out
}

func codePipelineCloneSliceAny(in []any) []any {
	out := make([]any, 0, len(in))
	for _, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out = append(out, codePipelineCloneMapAny(typed))
		case []any:
			out = append(out, codePipelineCloneSliceAny(typed))
		default:
			out = append(out, value)
		}
	}
	return out
}

func stringField(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	asString, ok := value.(string)
	if !ok {
		return ""
	}
	return asString
}
