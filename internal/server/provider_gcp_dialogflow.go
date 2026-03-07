package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPDialogflowRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPDialogflowPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.cloud.dialogflow.v2.") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPDialogflowSearchAgents(w, r, path) {
			return true
		}
		if handleGCPDialogflowGetAgent(w, path) {
			return true
		}
		if handleGCPDialogflowGetValidationResult(w, path) {
			return true
		}
		if handleGCPDialogflowListIntents(w, r, path) {
			return true
		}
		if handleGCPDialogflowGetIntent(w, path) {
			return true
		}
		if handleGCPDialogflowListOperations(w, r, path) {
			return true
		}
		if handleGCPDialogflowGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPDialogflowTrainAgent(w, path) {
			return true
		}
		if handleGCPDialogflowCreateIntent(w, r, path) {
			return true
		}
		if handleGCPDialogflowDetectIntent(w, r, path) {
			return true
		}
		if handleGCPDialogflowCancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPDialogflowSetAgent(w, r, path) {
			return true
		}
		if handleGCPDialogflowUpdateIntent(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPDialogflowDeleteIntent(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPDialogflowPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.dialogflow.v2.") {
		return true
	}
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || project == "" || len(tail) == 0 {
		return false
	}
	root := normalizeGCPDialogflowActionSegment(tail[0])
	switch root {
	case "agent", "agent:search", "agent:train", "sessions", "operations":
		return true
	default:
		return false
	}
}

func handleGCPDialogflowSearchAgents(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 1 || tail[0] != "agent:search" {
		return false
	}
	pageSize, start, valid := parseGCPDialogflowPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDialogflowAgent(project)}
	return respondGCPDialogflowList(w, "agents", items, pageSize, start, path)
}

func handleGCPDialogflowGetAgent(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 1 || tail[0] != "agent" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDialogflowAgent(project))
	return true
}

func handleGCPDialogflowSetAgent(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 1 || tail[0] != "agent" {
		return false
	}
	body, valid := decodeGCPDialogflowJSONBody(w, r, path)
	if !valid {
		return true
	}
	agent := gcpDialogflowBodyMap(body, "agent")
	if len(agent) == 0 {
		respondGCPDialogflowInvalidArgument(w, path, "agent is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDialogflowAgent(project))
	return true
}

func handleGCPDialogflowGetValidationResult(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 2 || tail[0] != "agent" || tail[1] != "validationResult" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":             fmt.Sprintf("projects/%s/agent/validationResult", project),
		"validationErrors": []any{},
	})
	return true
}

func handleGCPDialogflowTrainAgent(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 1 || tail[0] != "agent:train" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDialogflowOperation(project, "train-agent"))
	return true
}

func handleGCPDialogflowListIntents(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 2 || tail[0] != "agent" || tail[1] != "intents" {
		return false
	}
	pageSize, start, valid := parseGCPDialogflowPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDialogflowIntent(project, "intent-1")}
	return respondGCPDialogflowList(w, "intents", items, pageSize, start, path)
}

func handleGCPDialogflowCreateIntent(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 2 || tail[0] != "agent" || tail[1] != "intents" {
		return false
	}
	body, valid := decodeGCPDialogflowJSONBody(w, r, path)
	if !valid {
		return true
	}
	intent := gcpDialogflowBodyMap(body, "intent")
	if len(intent) == 0 {
		respondGCPDialogflowInvalidArgument(w, path, "intent is required")
		return true
	}
	intentID := strings.TrimSpace(stringFromMap(intent, "displayName"))
	if intentID == "" {
		intentID = "intent-1"
	}
	respondJSON(w, http.StatusOK, gcpDialogflowIntent(project, intentID))
	return true
}

func handleGCPDialogflowGetIntent(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 3 || tail[0] != "agent" || tail[1] != "intents" || strings.TrimSpace(tail[2]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDialogflowIntent(project, tail[2]))
	return true
}

func handleGCPDialogflowUpdateIntent(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 3 || tail[0] != "agent" || tail[1] != "intents" || strings.TrimSpace(tail[2]) == "" {
		return false
	}
	body, valid := decodeGCPDialogflowJSONBody(w, r, path)
	if !valid {
		return true
	}
	intent := gcpDialogflowBodyMap(body, "intent")
	if len(intent) == 0 {
		respondGCPDialogflowInvalidArgument(w, path, "intent is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDialogflowIntent(project, tail[2]))
	return true
}

func handleGCPDialogflowDeleteIntent(w http.ResponseWriter, path string) bool {
	_, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 3 || tail[0] != "agent" || tail[1] != "intents" || strings.TrimSpace(tail[2]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDialogflowDetectIntent(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 3 || tail[0] != "agent" || tail[1] != "sessions" {
		return false
	}
	sessionAction := normalizeGCPDialogflowActionSegment(tail[2])
	sessionID, action, hasAction := strings.Cut(sessionAction, ":")
	if !hasAction || strings.TrimSpace(sessionID) == "" || action != "detectIntent" {
		return false
	}
	body, valid := decodeGCPDialogflowJSONBody(w, r, path)
	if !valid {
		return true
	}
	if _, ok := body["queryInput"]; !ok {
		respondGCPDialogflowInvalidArgument(w, path, "queryInput is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"responseId": "resp-1",
		"queryResult": map[string]any{
			"queryText":                 "hello from stackyard",
			"intentDetectionConfidence": 0.88,
			"intent":                    gcpDialogflowIntent(project, "intent-1"),
			"languageCode":              "en",
		},
		"webhookStatus": map[string]any{},
	})
	return true
}

func handleGCPDialogflowListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 1 || tail[0] != "operations" {
		return false
	}
	pageSize, start, valid := parseGCPDialogflowPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDialogflowOperation(project, "op-1")}
	return respondGCPDialogflowList(w, "operations", items, pageSize, start, path)
}

func handleGCPDialogflowGetOperation(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDialogflowOperation(project, tail[1]))
	return true
}

func handleGCPDialogflowCancelOperation(w http.ResponseWriter, path string) bool {
	_, tail, ok := parseGCPDialogflowProjectTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	opAction := normalizeGCPDialogflowActionSegment(tail[1])
	operationID, action, hasAction := strings.Cut(opAction, ":")
	if !hasAction || strings.TrimSpace(operationID) == "" || action != "cancel" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPDialogflowProjectTail(path string) (project string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" {
		return "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", nil, false
	}
	tail = parts[4:]
	return project, tail, len(tail) > 0
}

func parseGCPDialogflowPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPDialogflowInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPDialogflowInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPDialogflowList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPDialogflowInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPDialogflowJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPDialogflowInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpDialogflowBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func normalizeGCPDialogflowActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func gcpDialogflowAgent(project string) map[string]any {
	return map[string]any{
		"parent":              fmt.Sprintf("projects/%s", project),
		"displayName":         "Stackyard Agent",
		"defaultLanguageCode": "en",
		"timeZone":            "America/New_York",
		"description":         "Dialogflow agent from Stackyard example",
	}
}

func gcpDialogflowIntent(project, intentID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/agent/intents/%s", project, intentID),
		"displayName": intentID,
		"priority":    500000,
	}
}

func gcpDialogflowOperation(project, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/operations/%s", project, operationID),
		"done": true,
	}
}

func respondGCPDialogflowInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
