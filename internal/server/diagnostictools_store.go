package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type diagnosticToolsStore struct {
	mu sync.Mutex

	nextExecution int64

	tools      map[string]map[string]any
	executions map[string]map[string]any
	outputs    map[string]map[string]any
	tags       map[string]map[string]string
}

func newDiagnosticToolsStore() *diagnosticToolsStore {
	s := &diagnosticToolsStore{
		nextExecution: 2,
		tools:         map[string]map[string]any{},
		executions:    map[string]map[string]any{},
		outputs:       map[string]map[string]any{},
		tags:          map[string]map[string]string{},
	}

	s.ensureToolLocked("EC2SystemsManager")
	s.ensureExecutionLocked("e-000001", map[string]any{
		"toolId":        "EC2SystemsManager",
		"toolVersionId": "1.0.0",
		"targetRegions": []any{"us-east-1"},
		"storageRegion": "us-east-1",
		"roleArn":       "arn:aws:iam::123456789012:role/stackyard-diagnostic-tools",
	})
	s.tags["e-000001"] = map[string]string{"stackyard": "true"}
	s.syncExecutionTagsLocked("e-000001")
	return s
}

func (s *diagnosticToolsStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	executionID := diagnosticToolsPayloadString(payload, "executionId", "e-000001")
	toolID := diagnosticToolsPayloadString(payload, "toolId", "EC2SystemsManager")
	identifier := diagnosticToolsPayloadString(payload, "identifier", executionID)

	switch action {
	case "StartExecution":
		if strings.TrimSpace(executionID) == "" || executionID == "e-000001" {
			executionID = s.nextExecutionIDLocked()
		}
		exec := s.ensureExecutionLocked(executionID, payload)
		if tags := diagnosticToolsExtractTags(payload); len(tags) > 0 {
			s.ensureTagsLocked(executionID)
			for k, v := range tags {
				s.tags[executionID][k] = v
			}
			s.syncExecutionTagsLocked(executionID)
			exec = s.executions[executionID]
		}
		return map[string]any{"execution": diagnosticToolsCloneMap(exec)}

	case "GetExecution":
		exec := s.ensureExecutionLocked(executionID, payload)
		return map[string]any{"execution": diagnosticToolsCloneMap(exec)}

	case "GetExecutionOutput":
		exec := s.ensureExecutionLocked(executionID, payload)
		output := s.ensureOutputLocked(executionID, exec)
		return diagnosticToolsCloneMap(output)

	case "ListExecutions":
		items := make([]map[string]any, 0, len(s.executions))
		ids := make([]string, 0, len(s.executions))
		for id := range s.executions {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			items = append(items, diagnosticToolsCloneMap(s.executions[id]))
		}
		return map[string]any{"executions": diagnosticToolsMapSliceToAny(items), "nextToken": ""}

	case "GetTool":
		tool := s.ensureToolLocked(toolID)
		return map[string]any{"tool": diagnosticToolsCloneMap(tool)}

	case "ListTools":
		items := make([]map[string]any, 0, len(s.tools))
		ids := make([]string, 0, len(s.tools))
		for id := range s.tools {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			items = append(items, diagnosticToolsCloneMap(s.tools[id]))
		}
		return map[string]any{"tools": diagnosticToolsMapSliceToAny(items), "nextToken": ""}

	case "TagResource":
		if strings.TrimSpace(identifier) == "" {
			identifier = "e-000001"
		}
		s.ensureExecutionLocked(identifier, map[string]any{"executionId": identifier})
		s.ensureTagsLocked(identifier)
		for k, v := range diagnosticToolsExtractTags(payload) {
			s.tags[identifier][k] = v
		}
		s.syncExecutionTagsLocked(identifier)
		return map[string]any{}

	case "UntagResource":
		if strings.TrimSpace(identifier) == "" {
			identifier = "e-000001"
		}
		s.ensureExecutionLocked(identifier, map[string]any{"executionId": identifier})
		s.ensureTagsLocked(identifier)
		for _, key := range diagnosticToolsExtractTagKeys(payload) {
			delete(s.tags[identifier], key)
		}
		s.syncExecutionTagsLocked(identifier)
		return map[string]any{}

	case "ListTagsForResource":
		if strings.TrimSpace(identifier) == "" {
			identifier = "e-000001"
		}
		s.ensureExecutionLocked(identifier, map[string]any{"executionId": identifier})
		s.ensureTagsLocked(identifier)
		return map[string]any{"tags": diagnosticToolsTagsList(s.tags[identifier])}
	}

	return map[string]any{}
}

func (s *diagnosticToolsStore) ensureExecutionLocked(executionID string, payload map[string]any) map[string]any {
	executionID = strings.TrimSpace(executionID)
	if executionID == "" {
		executionID = "e-000001"
	}
	if existing, ok := s.executions[executionID]; ok {
		return existing
	}

	toolID := diagnosticToolsPayloadString(payload, "toolId", "EC2SystemsManager")
	toolVersionID := diagnosticToolsPayloadString(payload, "toolVersionId", "1.0.0")
	roleArn := diagnosticToolsPayloadString(payload, "roleArn", "arn:aws:iam::123456789012:role/stackyard-diagnostic-tools")
	storageRegion := diagnosticToolsPayloadString(payload, "storageRegion", "us-east-1")
	targetRegions := diagnosticToolsExtractStringList(payload["targetRegions"])
	if len(targetRegions) == 0 {
		targetRegions = []string{"us-east-1"}
	}

	exec := map[string]any{
		"creationTime":  time.Now().UTC().Unix(),
		"executionId":   executionID,
		"requestState":  "SUBMITTED",
		"requesterArn":  "arn:aws:iam::123456789012:user/stackyard",
		"requesterId":   "stackyard",
		"roleArn":       roleArn,
		"status":        "CREATED",
		"storageRegion": storageRegion,
		"tags":          []any{},
		"targetRegions": diagnosticToolsStringSliceToAny(targetRegions),
		"toolId":        toolID,
		"toolVersionId": toolVersionID,
	}
	s.executions[executionID] = exec
	s.ensureOutputLocked(executionID, exec)
	s.ensureTagsLocked(executionID)
	s.syncExecutionTagsLocked(executionID)
	return exec
}

func (s *diagnosticToolsStore) ensureOutputLocked(executionID string, execution map[string]any) map[string]any {
	if existing, ok := s.outputs[executionID]; ok {
		return existing
	}

	output := map[string]any{
		"executionId": executionID,
		"mimeType":    "application/json",
		"output":      fmt.Sprintf("{\"executionId\":\"%s\",\"status\":\"SUCCEEDED\"}", executionID),
		"status":      execution["status"],
	}
	s.outputs[executionID] = output
	return output
}

func (s *diagnosticToolsStore) ensureToolLocked(toolID string) map[string]any {
	toolID = strings.TrimSpace(toolID)
	if toolID == "" {
		toolID = "EC2SystemsManager"
	}
	if existing, ok := s.tools[toolID]; ok {
		return existing
	}
	now := time.Now().UTC().Unix()
	tool := map[string]any{
		"toolId":         toolID,
		"displayName":    toolID,
		"description":    "Stackyard diagnostic tool",
		"createdAt":      now,
		"lastUpdatedAt":  now,
		"defaultVersion": "1.0.0",
		"versions": []any{
			map[string]any{
				"versionId":     "1.0.0",
				"status":        "ACTIVE",
				"releaseNotes":  "initial version",
				"lastUpdatedAt": now,
			},
		},
	}
	s.tools[toolID] = tool
	return tool
}

func (s *diagnosticToolsStore) nextExecutionIDLocked() string {
	id := fmt.Sprintf("e-%06d", s.nextExecution)
	s.nextExecution++
	return id
}

func (s *diagnosticToolsStore) ensureTagsLocked(identifier string) {
	if _, ok := s.tags[identifier]; ok {
		return
	}
	s.tags[identifier] = map[string]string{}
}

func (s *diagnosticToolsStore) syncExecutionTagsLocked(executionID string) {
	exec, ok := s.executions[executionID]
	if !ok {
		return
	}
	exec["tags"] = diagnosticToolsTagsList(s.tags[executionID])
}

func diagnosticToolsPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return fallback
	}
	value, ok := raw.(string)
	if !ok {
		return fallback
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func diagnosticToolsExtractStringList(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func diagnosticToolsExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	raw, ok := payload["tags"]
	if !ok || raw == nil {
		return out
	}

	switch tags := raw.(type) {
	case map[string]any:
		for k, v := range tags {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	case []any:
		for _, item := range tags {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := strings.TrimSpace(fmt.Sprintf("%v", m["key"]))
			if key == "" || key == "<nil>" {
				key = strings.TrimSpace(fmt.Sprintf("%v", m["Key"]))
			}
			if key == "" || key == "<nil>" {
				continue
			}
			value := strings.TrimSpace(fmt.Sprintf("%v", m["value"]))
			if value == "" || value == "<nil>" {
				value = strings.TrimSpace(fmt.Sprintf("%v", m["Value"]))
			}
			if value == "<nil>" {
				value = ""
			}
			out[key] = value
		}
	case []map[string]any:
		for _, item := range tags {
			key := strings.TrimSpace(fmt.Sprintf("%v", item["key"]))
			if key == "" || key == "<nil>" {
				key = strings.TrimSpace(fmt.Sprintf("%v", item["Key"]))
			}
			if key == "" || key == "<nil>" {
				continue
			}
			value := strings.TrimSpace(fmt.Sprintf("%v", item["value"]))
			if value == "" || value == "<nil>" {
				value = strings.TrimSpace(fmt.Sprintf("%v", item["Value"]))
			}
			if value == "<nil>" {
				value = ""
			}
			out[key] = value
		}
	case string:
		tag := strings.TrimSpace(tags)
		if tag != "" {
			out[tag] = "true"
		}
	}

	return out
}

func diagnosticToolsExtractTagKeys(payload map[string]any) []string {
	out := []string{}
	if payload == nil {
		return out
	}
	raw, ok := payload["tagKeys"]
	if !ok || raw == nil {
		return out
	}

	switch keys := raw.(type) {
	case []any:
		for _, item := range keys {
			s, ok := item.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
	case string:
		for _, part := range strings.Split(keys, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}

	return out
}

func diagnosticToolsTagsList(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"key": key, "value": tags[key]})
	}
	return out
}

func diagnosticToolsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func diagnosticToolsMapSliceToAny(in []map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, diagnosticToolsCloneMap(item))
	}
	return out
}

func diagnosticToolsStringSliceToAny(in []string) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, item)
	}
	return out
}
