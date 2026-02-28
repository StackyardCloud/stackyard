package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type bedrockAgentCoreDataStore struct {
	mu sync.Mutex

	nextRecordID int64
	nextEventID  int64
	nextSession  int64
	nextJobID    int64

	memoryRecords map[string]map[string]any
	events        map[string]map[string]any
	sessions      map[string]map[string]any
	browser       map[string]map[string]any
	code          map[string]map[string]any
	extractions   map[string]map[string]any
}

func newBedrockAgentCoreDataStore() *bedrockAgentCoreDataStore {
	s := &bedrockAgentCoreDataStore{
		nextRecordID:  2,
		nextEventID:   2,
		nextSession:   2,
		nextJobID:     2,
		memoryRecords: map[string]map[string]any{},
		events:        map[string]map[string]any{},
		sessions:      map[string]map[string]any{},
		browser:       map[string]map[string]any{},
		code:          map[string]map[string]any{},
		extractions:   map[string]map[string]any{},
	}
	s.seedLocked(time.Now().UTC())
	return s
}

func (s *bedrockAgentCoreDataStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.seedLocked(now)

	ctx := bagcdMergeMaps(payload, pathParams, query)
	memoryID := bagcdString(ctx, "memoryId", "memory-000001")
	recordID := bagcdString(ctx, "memoryRecordId", "record-000001")
	eventID := bagcdString(ctx, "eventId", "event-000001")
	actorID := bagcdString(ctx, "actorId", "actor-000001")
	sessionID := bagcdString(ctx, "sessionId", "session-000001")
	browserID := bagcdString(ctx, "browserIdentifier", "browser-000001")
	codeID := bagcdString(ctx, "codeInterpreterIdentifier", "code-000001")
	profileID := bagcdString(ctx, "profileIdentifier", "profile-000001")
	runtimeARN := bagcdString(ctx, "agentRuntimeArn", "arn:aws:bedrock-agentcore:us-east-1:123456789012:runtime/stackyard-runtime")
	qualifier := bagcdString(ctx, "qualifier", "LATEST")
	evaluatorID := bagcdString(ctx, "evaluatorId", "evaluator-000001")

	recordKey := memoryID + "|" + recordID
	eventKey := memoryID + "|" + actorID + "|" + sessionID + "|" + eventID

	s.ensureMemoryRecordLocked(recordKey, memoryID, recordID, now)
	s.ensureEventLocked(eventKey, memoryID, actorID, sessionID, eventID, now)
	s.ensureSessionLocked(sessionID, actorID, memoryID, now)
	s.ensureBrowserSessionLocked(browserID, sessionID, now)
	s.ensureCodeSessionLocked(codeID, sessionID, now)

	switch action {
	case "BatchCreateMemoryRecords":
		id := fmt.Sprintf("record-%06d", s.nextRecordID)
		s.nextRecordID++
		key := memoryID + "|" + id
		rec := s.ensureMemoryRecordLocked(key, memoryID, id, now)
		return map[string]any{"memoryRecords": []any{bagcdCloneMap(rec)}, "errors": []any{}}
	case "BatchDeleteMemoryRecords":
		delete(s.memoryRecords, recordKey)
		return map[string]any{"deletedMemoryRecordIds": []any{recordID}, "errors": []any{}}
	case "BatchUpdateMemoryRecords":
		rec := s.ensureMemoryRecordLocked(recordKey, memoryID, recordID, now)
		rec["updatedAt"] = now.Format(time.RFC3339)
		return map[string]any{"memoryRecords": []any{bagcdCloneMap(rec)}, "errors": []any{}}
	case "CompleteResourceTokenAuth":
		return map[string]any{"status": "COMPLETED", "token": "stackyard-resource-token"}
	case "CreateEvent":
		id := fmt.Sprintf("event-%06d", s.nextEventID)
		s.nextEventID++
		key := memoryID + "|" + actorID + "|" + sessionID + "|" + id
		event := s.ensureEventLocked(key, memoryID, actorID, sessionID, id, now)
		return map[string]any{"event": bagcdCloneMap(event)}
	case "DeleteEvent":
		delete(s.events, eventKey)
		return map[string]any{}
	case "DeleteMemoryRecord":
		delete(s.memoryRecords, recordKey)
		return map[string]any{}
	case "Evaluate":
		return map[string]any{
			"evaluatorId": evaluatorID,
			"result":      map[string]any{"status": "SUCCEEDED", "score": 1.0},
		}
	case "GetAgentCard":
		return map[string]any{
			"agentRuntimeArn": runtimeARN,
			"qualifier":       qualifier,
			"version":         "1.0",
			"name":            "stackyard-agent",
		}
	case "GetBrowserSession":
		return map[string]any{"session": bagcdCloneMap(s.ensureBrowserSessionLocked(browserID, sessionID, now))}
	case "GetCodeInterpreterSession":
		return map[string]any{"session": bagcdCloneMap(s.ensureCodeSessionLocked(codeID, sessionID, now))}
	case "GetEvent":
		return map[string]any{"event": bagcdCloneMap(s.ensureEventLocked(eventKey, memoryID, actorID, sessionID, eventID, now))}
	case "GetMemoryRecord":
		return map[string]any{"memoryRecord": bagcdCloneMap(s.ensureMemoryRecordLocked(recordKey, memoryID, recordID, now))}
	case "GetResourceApiKey":
		return map[string]any{"apiKey": "stackyard-api-key", "expiresAt": now.Add(1 * time.Hour).Format(time.RFC3339)}
	case "GetResourceOauth2Token":
		return map[string]any{"accessToken": "stackyard-oauth-token", "tokenType": "Bearer", "expiresIn": 3600}
	case "GetWorkloadAccessToken", "GetWorkloadAccessTokenForJWT", "GetWorkloadAccessTokenForUserId":
		return map[string]any{"accessToken": "stackyard-workload-token", "tokenType": "Bearer", "expiresIn": 3600}
	case "InvokeAgentRuntime":
		return map[string]any{
			"sessionId": sessionID,
			"output":    map[string]any{"content": []any{map[string]any{"text": "stackyard-runtime-output"}}},
		}
	case "InvokeCodeInterpreter":
		return map[string]any{"sessionId": sessionID, "result": map[string]any{"text": "stackyard-code-output"}}
	case "ListActors":
		return map[string]any{"actors": []any{map[string]any{"actorId": actorID, "memoryId": memoryID}}, "nextToken": ""}
	case "ListBrowserSessions":
		return map[string]any{"sessions": bagcdListByPrefix(s.browser, browserID+"|"), "nextToken": ""}
	case "ListCodeInterpreterSessions":
		return map[string]any{"sessions": bagcdListByPrefix(s.code, codeID+"|"), "nextToken": ""}
	case "ListEvents":
		return map[string]any{"events": bagcdListByPrefix(s.events, memoryID+"|"+actorID+"|"+sessionID+"|"), "nextToken": ""}
	case "ListMemoryExtractionJobs":
		return map[string]any{"extractionJobs": bagcdListByPrefix(s.extractions, memoryID+"|"), "nextToken": ""}
	case "ListMemoryRecords":
		return map[string]any{"memoryRecords": bagcdListByPrefix(s.memoryRecords, memoryID+"|"), "nextToken": ""}
	case "ListSessions":
		return map[string]any{"sessions": bagcdListByPrefix(s.sessions, ""), "nextToken": ""}
	case "RetrieveMemoryRecords":
		return map[string]any{"memoryRecords": bagcdListByPrefix(s.memoryRecords, memoryID+"|"), "nextToken": ""}
	case "SaveBrowserSessionProfile":
		return map[string]any{"profileIdentifier": profileID, "status": "SAVED"}
	case "StartBrowserSession":
		id := fmt.Sprintf("session-%06d", s.nextSession)
		s.nextSession++
		sess := s.ensureBrowserSessionLocked(browserID, id, now)
		return map[string]any{"session": bagcdCloneMap(sess)}
	case "StartCodeInterpreterSession":
		id := fmt.Sprintf("session-%06d", s.nextSession)
		s.nextSession++
		sess := s.ensureCodeSessionLocked(codeID, id, now)
		return map[string]any{"session": bagcdCloneMap(sess)}
	case "StartMemoryExtractionJob":
		jobID := fmt.Sprintf("job-%06d", s.nextJobID)
		s.nextJobID++
		key := memoryID + "|" + jobID
		job := map[string]any{"extractionJobId": jobID, "memoryId": memoryID, "status": "IN_PROGRESS", "startedAt": now.Format(time.RFC3339)}
		s.extractions[key] = job
		return map[string]any{"extractionJob": bagcdCloneMap(job)}
	case "StopBrowserSession":
		sess := s.ensureBrowserSessionLocked(browserID, sessionID, now)
		sess["status"] = "STOPPED"
		sess["stoppedAt"] = now.Format(time.RFC3339)
		return map[string]any{"session": bagcdCloneMap(sess)}
	case "StopCodeInterpreterSession":
		sess := s.ensureCodeSessionLocked(codeID, sessionID, now)
		sess["status"] = "STOPPED"
		sess["stoppedAt"] = now.Format(time.RFC3339)
		return map[string]any{"session": bagcdCloneMap(sess)}
	case "StopRuntimeSession":
		sess := s.ensureSessionLocked(sessionID, actorID, memoryID, now)
		sess["status"] = "STOPPED"
		sess["stoppedAt"] = now.Format(time.RFC3339)
		return map[string]any{"session": bagcdCloneMap(sess)}
	case "UpdateBrowserStream":
		return map[string]any{"sessionId": sessionID, "status": "UPDATED"}
	}

	return map[string]any{}
}

func (s *bedrockAgentCoreDataStore) seedLocked(now time.Time) {
	s.ensureMemoryRecordLocked("memory-000001|record-000001", "memory-000001", "record-000001", now)
	s.ensureEventLocked("memory-000001|actor-000001|session-000001|event-000001", "memory-000001", "actor-000001", "session-000001", "event-000001", now)
	s.ensureSessionLocked("session-000001", "actor-000001", "memory-000001", now)
	s.ensureBrowserSessionLocked("browser-000001", "session-000001", now)
	s.ensureCodeSessionLocked("code-000001", "session-000001", now)
	if len(s.extractions) == 0 {
		s.extractions["memory-000001|job-000001"] = map[string]any{
			"extractionJobId": "job-000001",
			"memoryId":        "memory-000001",
			"status":          "SUCCEEDED",
			"startedAt":       now.Add(-5 * time.Minute).Format(time.RFC3339),
		}
	}
}

func (s *bedrockAgentCoreDataStore) ensureMemoryRecordLocked(key, memoryID, recordID string, now time.Time) map[string]any {
	if rec := s.memoryRecords[key]; rec != nil {
		return rec
	}
	rec := map[string]any{
		"memoryRecordId": recordID,
		"memoryId":       memoryID,
		"content":        "stackyard-memory-content",
		"createdAt":      now.Format(time.RFC3339),
		"updatedAt":      now.Format(time.RFC3339),
	}
	s.memoryRecords[key] = rec
	return rec
}

func (s *bedrockAgentCoreDataStore) ensureEventLocked(key, memoryID, actorID, sessionID, eventID string, now time.Time) map[string]any {
	if evt := s.events[key]; evt != nil {
		return evt
	}
	evt := map[string]any{
		"eventId":     eventID,
		"memoryId":    memoryID,
		"actorId":     actorID,
		"sessionId":   sessionID,
		"createdAt":   now.Format(time.RFC3339),
		"contentType": "TEXT",
	}
	s.events[key] = evt
	return evt
}

func (s *bedrockAgentCoreDataStore) ensureSessionLocked(sessionID, actorID, memoryID string, now time.Time) map[string]any {
	if sess := s.sessions[sessionID]; sess != nil {
		return sess
	}
	sess := map[string]any{
		"sessionId": sessionID,
		"actorId":   actorID,
		"memoryId":  memoryID,
		"status":    "ACTIVE",
		"startedAt": now.Format(time.RFC3339),
	}
	s.sessions[sessionID] = sess
	return sess
}

func (s *bedrockAgentCoreDataStore) ensureBrowserSessionLocked(browserID, sessionID string, now time.Time) map[string]any {
	key := browserID + "|" + sessionID
	if sess := s.browser[key]; sess != nil {
		return sess
	}
	sess := map[string]any{
		"sessionId":         sessionID,
		"browserIdentifier": browserID,
		"status":            "ACTIVE",
		"startedAt":         now.Format(time.RFC3339),
	}
	s.browser[key] = sess
	return sess
}

func (s *bedrockAgentCoreDataStore) ensureCodeSessionLocked(codeID, sessionID string, now time.Time) map[string]any {
	key := codeID + "|" + sessionID
	if sess := s.code[key]; sess != nil {
		return sess
	}
	sess := map[string]any{
		"sessionId":                 sessionID,
		"codeInterpreterIdentifier": codeID,
		"status":                    "ACTIVE",
		"startedAt":                 now.Format(time.RFC3339),
	}
	s.code[key] = sess
	return sess
}

func bagcdMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for k, v := range payload {
		out[k] = v
	}
	for k, v := range pathParams {
		out[k] = v
	}
	for k, values := range query {
		if len(values) > 0 {
			out[k] = values[len(values)-1]
		}
	}
	return out
}

func bagcdString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		value := strings.TrimSpace(fmt.Sprint(v))
		if value != "" {
			return value
		}
	}
	return def
}

func bagcdListByPrefix(items map[string]map[string]any, prefix string) []any {
	keys := make([]string, 0, len(items))
	for key := range items {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, bagcdCloneMap(items[key]))
	}
	return out
}

func bagcdCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = bagcdCloneMap(typed)
		case []any:
			items := make([]any, 0, len(typed))
			for _, item := range typed {
				if m, ok := item.(map[string]any); ok {
					items = append(items, bagcdCloneMap(m))
				} else {
					items = append(items, item)
				}
			}
			out[k] = items
		default:
			out[k] = v
		}
	}
	return out
}
