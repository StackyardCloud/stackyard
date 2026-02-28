package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type novaActStore struct {
	mu sync.Mutex

	nextWorkflowRunID int64
	nextSessionID     int64
	nextActID         int64

	workflowDefinitions map[string]map[string]any
	workflowRuns        map[string]map[string]any
	sessions            map[string]map[string]any
	acts                map[string]map[string]any
	models              map[string]map[string]any
}

func newNovaActStore() *novaActStore {
	s := &novaActStore{
		nextWorkflowRunID:   2,
		nextSessionID:       2,
		nextActID:           2,
		workflowDefinitions: map[string]map[string]any{},
		workflowRuns:        map[string]map[string]any{},
		sessions:            map[string]map[string]any{},
		acts:                map[string]map[string]any{},
		models:              map[string]map[string]any{},
	}
	s.seedLocked(time.Now().UTC())
	return s
}

func (s *novaActStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.seedLocked(now)

	ctx := novaActMergePayload(payload, pathParams, query)
	workflowDefinitionName := novaActString(ctx, "workflowDefinitionName", "stackyard-workflow")
	workflowRunID := novaActString(ctx, "workflowRunId", "workflow-run-000001")
	sessionID := novaActString(ctx, "sessionId", "session-000001")
	actID := novaActString(ctx, "actId", "act-000001")

	switch action {
	case "CreateWorkflowDefinition":
		name := novaActString(payload, "workflowDefinitionName", "")
		if name == "" {
			name = novaActString(payload, "name", workflowDefinitionName)
		}
		def := s.ensureWorkflowDefinitionLocked(name, now)
		if description := novaActString(payload, "description", ""); description != "" {
			def["description"] = description
		}
		def["status"] = "ACTIVE"
		def["lastUpdatedAt"] = now.Format(time.RFC3339)
		return map[string]any{"workflowDefinition": novaActCloneMap(def)}

	case "DeleteWorkflowDefinition":
		s.deleteWorkflowDefinitionLocked(workflowDefinitionName)
		return map[string]any{}

	case "GetWorkflowDefinition":
		def := s.ensureWorkflowDefinitionLocked(workflowDefinitionName, now)
		return map[string]any{"workflowDefinition": novaActCloneMap(def)}

	case "ListWorkflowDefinitions":
		return map[string]any{
			"workflowDefinitionSummaries": s.listWorkflowDefinitionsLocked(),
			"workflowDefinitions":         s.listWorkflowDefinitionsLocked(),
			"nextToken":                   "",
		}

	case "CreateWorkflowRun":
		runID := novaActString(payload, "workflowRunId", workflowRunID)
		if strings.TrimSpace(runID) == "" {
			runID = fmt.Sprintf("workflow-run-%06d", s.nextWorkflowRunID)
			s.nextWorkflowRunID++
		}
		run := s.ensureWorkflowRunLocked(workflowDefinitionName, runID, now)
		run["status"] = novaActString(payload, "status", "IN_PROGRESS")
		run["lastUpdatedAt"] = now.Format(time.RFC3339)
		return map[string]any{"workflowRun": novaActCloneMap(run)}

	case "DeleteWorkflowRun":
		s.deleteWorkflowRunLocked(workflowDefinitionName, workflowRunID)
		return map[string]any{}

	case "GetWorkflowRun":
		run := s.ensureWorkflowRunLocked(workflowDefinitionName, workflowRunID, now)
		return map[string]any{"workflowRun": novaActCloneMap(run)}

	case "UpdateWorkflowRun":
		run := s.ensureWorkflowRunLocked(workflowDefinitionName, workflowRunID, now)
		status := novaActString(payload, "status", "")
		if status == "" {
			status = "SUCCEEDED"
		}
		run["status"] = status
		run["lastUpdatedAt"] = now.Format(time.RFC3339)
		return map[string]any{"workflowRun": novaActCloneMap(run)}

	case "ListWorkflowRuns":
		return map[string]any{
			"workflowRunSummaries": s.listWorkflowRunsLocked(workflowDefinitionName),
			"workflowRuns":         s.listWorkflowRunsLocked(workflowDefinitionName),
			"nextToken":            "",
		}

	case "CreateSession":
		sessionIdentifier := novaActString(payload, "sessionId", sessionID)
		if strings.TrimSpace(sessionIdentifier) == "" {
			sessionIdentifier = fmt.Sprintf("session-%06d", s.nextSessionID)
			s.nextSessionID++
		}
		session := s.ensureSessionLocked(workflowDefinitionName, workflowRunID, sessionIdentifier, now)
		session["status"] = "ACTIVE"
		session["lastUpdatedAt"] = now.Format(time.RFC3339)
		return map[string]any{"session": novaActCloneMap(session)}

	case "ListSessions":
		return map[string]any{
			"sessionSummaries": s.listSessionsLocked(workflowDefinitionName, workflowRunID),
			"sessions":         s.listSessionsLocked(workflowDefinitionName, workflowRunID),
			"nextToken":        "",
		}

	case "CreateAct":
		actIdentifier := novaActString(payload, "actId", actID)
		if strings.TrimSpace(actIdentifier) == "" {
			actIdentifier = fmt.Sprintf("act-%06d", s.nextActID)
			s.nextActID++
		}
		act := s.ensureActLocked(workflowDefinitionName, workflowRunID, sessionID, actIdentifier, now)
		act["status"] = "CREATED"
		act["lastUpdatedAt"] = now.Format(time.RFC3339)
		if name := novaActString(payload, "name", ""); name != "" {
			act["name"] = name
		}
		if toolSpec, ok := payload["toolSpec"].(map[string]any); ok {
			act["toolSpec"] = novaActCloneMap(toolSpec)
		}
		return map[string]any{"act": novaActCloneMap(act)}

	case "UpdateAct":
		act := s.ensureActLocked(workflowDefinitionName, workflowRunID, sessionID, actID, now)
		status := novaActString(payload, "status", "UPDATED")
		act["status"] = status
		act["lastUpdatedAt"] = now.Format(time.RFC3339)
		if name := novaActString(payload, "name", ""); name != "" {
			act["name"] = name
		}
		return map[string]any{"act": novaActCloneMap(act)}

	case "ListActs":
		return map[string]any{
			"actSummaries": s.listActsLocked(workflowDefinitionName, workflowRunID, sessionID),
			"acts":         s.listActsLocked(workflowDefinitionName, workflowRunID, sessionID),
			"nextToken":    "",
		}

	case "InvokeActStep":
		act := s.ensureActLocked(workflowDefinitionName, workflowRunID, sessionID, actID, now)
		act["status"] = "COMPLETED"
		act["lastInvokedAt"] = now.Format(time.RFC3339)
		step := map[string]any{}
		if candidate, ok := payload["step"].(map[string]any); ok {
			step = novaActCloneMap(candidate)
		}
		return map[string]any{
			"actId":  actID,
			"status": "SUCCEEDED",
			"traceLocation": map[string]any{
				"type": "INLINE",
				"path": "stackyard://nova-act/traces/" + workflowDefinitionName + "/" + workflowRunID + "/" + sessionID + "/" + actID,
			},
			"callResult": map[string]any{
				"status": "SUCCEEDED",
				"content": map[string]any{
					"text": "stackyard-step-output",
				},
				"step": step,
			},
			"act": novaActCloneMap(act),
		}

	case "ListModels":
		version := novaActString(ctx, "clientCompatibilityVersion", "1.0")
		return map[string]any{
			"modelSummaries": s.listModelsLocked(),
			"models":         s.listModelsLocked(),
			"compatibilityInformation": map[string]any{
				"clientCompatibilityVersion": version,
				"status":                     "COMPATIBLE",
			},
			"nextToken": "",
		}
	}

	return map[string]any{}
}

func (s *novaActStore) seedLocked(now time.Time) {
	s.ensureWorkflowDefinitionLocked("stackyard-workflow", now)
	s.ensureWorkflowRunLocked("stackyard-workflow", "workflow-run-000001", now)
	s.ensureSessionLocked("stackyard-workflow", "workflow-run-000001", "session-000001", now)
	s.ensureActLocked("stackyard-workflow", "workflow-run-000001", "session-000001", "act-000001", now)

	if len(s.models) == 0 {
		s.models["nova-act-small"] = map[string]any{
			"modelId":         "nova-act-small",
			"modelArn":        "arn:aws:nova-act:us-east-1::model/nova-act-small",
			"modelAlias":      "small",
			"provider":        "Amazon",
			"lifecycle":       "ACTIVE",
			"lastUpdatedAt":   now.Format(time.RFC3339),
			"supportsTracing": true,
		}
		s.models["nova-act-large"] = map[string]any{
			"modelId":         "nova-act-large",
			"modelArn":        "arn:aws:nova-act:us-east-1::model/nova-act-large",
			"modelAlias":      "large",
			"provider":        "Amazon",
			"lifecycle":       "ACTIVE",
			"lastUpdatedAt":   now.Format(time.RFC3339),
			"supportsTracing": true,
		}
	}
}

func (s *novaActStore) ensureWorkflowDefinitionLocked(name string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-workflow"
	}
	if existing := s.workflowDefinitions[name]; existing != nil {
		return existing
	}
	item := map[string]any{
		"workflowDefinitionName": name,
		"workflowDefinitionArn":  "arn:aws:nova-act:us-east-1:123456789012:workflow-definition/" + name,
		"status":                 "ACTIVE",
		"createdAt":              now.Format(time.RFC3339),
		"lastUpdatedAt":          now.Format(time.RFC3339),
		"workflowExportConfig": map[string]any{
			"format": "JSON",
		},
	}
	s.workflowDefinitions[name] = item
	return item
}

func (s *novaActStore) ensureWorkflowRunLocked(workflowDefinitionName, workflowRunID string, now time.Time) map[string]any {
	definition := s.ensureWorkflowDefinitionLocked(workflowDefinitionName, now)
	workflowRunID = strings.TrimSpace(workflowRunID)
	if workflowRunID == "" {
		workflowRunID = "workflow-run-000001"
	}
	key := novaActWorkflowRunKey(workflowDefinitionName, workflowRunID)
	if existing := s.workflowRuns[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"workflowDefinitionName": workflowDefinitionName,
		"workflowDefinitionArn":  novaActString(definition, "workflowDefinitionArn", ""),
		"workflowRunId":          workflowRunID,
		"workflowRunArn":         "arn:aws:nova-act:us-east-1:123456789012:workflow-definition/" + workflowDefinitionName + "/workflow-run/" + workflowRunID,
		"status":                 "IN_PROGRESS",
		"createdAt":              now.Format(time.RFC3339),
		"lastUpdatedAt":          now.Format(time.RFC3339),
	}
	s.workflowRuns[key] = item
	return item
}

func (s *novaActStore) ensureSessionLocked(workflowDefinitionName, workflowRunID, sessionID string, now time.Time) map[string]any {
	s.ensureWorkflowRunLocked(workflowDefinitionName, workflowRunID, now)
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = "session-000001"
	}
	key := novaActSessionKey(workflowDefinitionName, workflowRunID, sessionID)
	if existing := s.sessions[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"workflowDefinitionName": workflowDefinitionName,
		"workflowRunId":          workflowRunID,
		"sessionId":              sessionID,
		"status":                 "ACTIVE",
		"createdAt":              now.Format(time.RFC3339),
		"lastUpdatedAt":          now.Format(time.RFC3339),
	}
	s.sessions[key] = item
	return item
}

func (s *novaActStore) ensureActLocked(workflowDefinitionName, workflowRunID, sessionID, actID string, now time.Time) map[string]any {
	s.ensureSessionLocked(workflowDefinitionName, workflowRunID, sessionID, now)
	actID = strings.TrimSpace(actID)
	if actID == "" {
		actID = "act-000001"
	}
	key := novaActActKey(workflowDefinitionName, workflowRunID, sessionID, actID)
	if existing := s.acts[key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"workflowDefinitionName": workflowDefinitionName,
		"workflowRunId":          workflowRunID,
		"sessionId":              sessionID,
		"actId":                  actID,
		"name":                   actID,
		"status":                 "CREATED",
		"createdAt":              now.Format(time.RFC3339),
		"lastUpdatedAt":          now.Format(time.RFC3339),
	}
	s.acts[key] = item
	return item
}

func (s *novaActStore) deleteWorkflowDefinitionLocked(workflowDefinitionName string) {
	delete(s.workflowDefinitions, workflowDefinitionName)

	runPrefix := workflowDefinitionName + "|"
	for key := range s.workflowRuns {
		if strings.HasPrefix(key, runPrefix) {
			delete(s.workflowRuns, key)
		}
	}
	for key := range s.sessions {
		if strings.HasPrefix(key, runPrefix) {
			delete(s.sessions, key)
		}
	}
	for key := range s.acts {
		if strings.HasPrefix(key, runPrefix) {
			delete(s.acts, key)
		}
	}
}

func (s *novaActStore) deleteWorkflowRunLocked(workflowDefinitionName, workflowRunID string) {
	runPrefix := workflowDefinitionName + "|" + workflowRunID
	delete(s.workflowRuns, runPrefix)
	prefix := runPrefix + "|"
	for key := range s.sessions {
		if strings.HasPrefix(key, prefix) {
			delete(s.sessions, key)
		}
	}
	for key := range s.acts {
		if strings.HasPrefix(key, prefix) {
			delete(s.acts, key)
		}
	}
}

func (s *novaActStore) listWorkflowDefinitionsLocked() []any {
	names := make([]string, 0, len(s.workflowDefinitions))
	for name := range s.workflowDefinitions {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, novaActCloneMap(s.workflowDefinitions[name]))
	}
	return out
}

func (s *novaActStore) listWorkflowRunsLocked(workflowDefinitionName string) []any {
	prefix := workflowDefinitionName + "|"
	keys := make([]string, 0, len(s.workflowRuns))
	for key := range s.workflowRuns {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, novaActCloneMap(s.workflowRuns[key]))
	}
	return out
}

func (s *novaActStore) listSessionsLocked(workflowDefinitionName, workflowRunID string) []any {
	prefix := workflowDefinitionName + "|" + workflowRunID + "|"
	keys := make([]string, 0, len(s.sessions))
	for key := range s.sessions {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, novaActCloneMap(s.sessions[key]))
	}
	return out
}

func (s *novaActStore) listActsLocked(workflowDefinitionName, workflowRunID, sessionID string) []any {
	prefix := workflowDefinitionName + "|"
	if strings.TrimSpace(workflowRunID) != "" {
		prefix += workflowRunID + "|"
		if strings.TrimSpace(sessionID) != "" {
			prefix += sessionID + "|"
		}
	}
	keys := make([]string, 0, len(s.acts))
	for key := range s.acts {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, novaActCloneMap(s.acts[key]))
	}
	return out
}

func (s *novaActStore) listModelsLocked() []any {
	keys := make([]string, 0, len(s.models))
	for key := range s.models {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, novaActCloneMap(s.models[key]))
	}
	return out
}

func novaActMergePayload(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		out[key] = values[len(values)-1]
	}
	return out
}

func novaActString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	for candidate, value := range payload {
		if !strings.EqualFold(candidate, key) {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" {
			return text
		}
	}
	return def
}

func novaActCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = novaActCloneMap(typed)
		case []any:
			items := make([]any, 0, len(typed))
			for _, item := range typed {
				if nested, ok := item.(map[string]any); ok {
					items = append(items, novaActCloneMap(nested))
				} else {
					items = append(items, item)
				}
			}
			out[key] = items
		default:
			out[key] = value
		}
	}
	return out
}

func novaActWorkflowRunKey(workflowDefinitionName, workflowRunID string) string {
	return workflowDefinitionName + "|" + workflowRunID
}

func novaActSessionKey(workflowDefinitionName, workflowRunID, sessionID string) string {
	return workflowDefinitionName + "|" + workflowRunID + "|" + sessionID
}

func novaActActKey(workflowDefinitionName, workflowRunID, sessionID, actID string) string {
	return workflowDefinitionName + "|" + workflowRunID + "|" + sessionID + "|" + actID
}
