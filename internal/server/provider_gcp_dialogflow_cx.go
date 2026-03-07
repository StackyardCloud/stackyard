package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPDialogflowCXRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPDialogflowCXPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.cloud.dialogflow.cx.v3.") {
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
		if handleGCPDialogflowCXListAgents(w, r, path) {
			return true
		}
		if handleGCPDialogflowCXGetAgent(w, path) {
			return true
		}
		if handleGCPDialogflowCXGetAgentValidationResult(w, path) {
			return true
		}
		if handleGCPDialogflowCXListFlows(w, r, path) {
			return true
		}
		if handleGCPDialogflowCXGetFlow(w, path) {
			return true
		}
		if handleGCPDialogflowCXGetFlowValidationResult(w, path) {
			return true
		}
		if handleGCPDialogflowCXListOperations(w, r, path) {
			return true
		}
		if handleGCPDialogflowCXGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPDialogflowCXCreateAgent(w, r, path) {
			return true
		}
		if handleGCPDialogflowCXValidateAgent(w, path) {
			return true
		}
		if handleGCPDialogflowCXExportAgent(w, path) {
			return true
		}
		if handleGCPDialogflowCXRestoreAgent(w, path) {
			return true
		}
		if handleGCPDialogflowCXCreateFlow(w, r, path) {
			return true
		}
		if handleGCPDialogflowCXValidateFlow(w, path) {
			return true
		}
		if handleGCPDialogflowCXTrainFlow(w, path) {
			return true
		}
		if handleGCPDialogflowCXDetectIntent(w, r, path) {
			return true
		}
		if handleGCPDialogflowCXMatchIntent(w, r, path) {
			return true
		}
		if handleGCPDialogflowCXFulfillIntent(w, r, path) {
			return true
		}
		if handleGCPDialogflowCXCancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPDialogflowCXUpdateAgent(w, r, path) {
			return true
		}
		if handleGCPDialogflowCXUpdateFlow(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPDialogflowCXPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.dialogflow.cx.v3.") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v3/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}
	return strings.Contains(path, "/agents") ||
		strings.Contains(path, "/flows") ||
		strings.Contains(path, "/sessions") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, "/validationResult") ||
		strings.Contains(path, ":validate") ||
		strings.Contains(path, ":train") ||
		strings.Contains(path, ":detectIntent") ||
		strings.Contains(path, ":matchIntent") ||
		strings.Contains(path, ":fulfillIntent") ||
		strings.Contains(path, ":streamingDetectIntent") ||
		strings.Contains(path, ":export") ||
		strings.Contains(path, ":restore") ||
		strings.Contains(path, ":cancel")
}

func handleGCPDialogflowCXListAgents(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "agents" {
		return false
	}
	pageSize, start, valid := parseGCPDialogflowCXPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDialogflowCXAgent(project, location, "agent-1")}
	return respondGCPDialogflowCXList(w, "agents", items, pageSize, start, path)
}

func handleGCPDialogflowCXGetAgent(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "agents" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDialogflowCXAgent(project, location, tail[1]))
	return true
}

func handleGCPDialogflowCXCreateAgent(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "agents" {
		return false
	}
	body, valid := decodeGCPDialogflowCXJSONBody(w, r, path)
	if !valid {
		return true
	}
	agent := gcpDialogflowCXBodyMap(body, "agent")
	if len(agent) == 0 {
		respondGCPDialogflowCXInvalidArgument(w, path, "agent is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDialogflowCXAgent(project, location, "agent-1"))
	return true
}

func handleGCPDialogflowCXUpdateAgent(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "agents" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	body, valid := decodeGCPDialogflowCXJSONBody(w, r, path)
	if !valid {
		return true
	}
	agent := gcpDialogflowCXBodyMap(body, "agent")
	if len(agent) == 0 {
		respondGCPDialogflowCXInvalidArgument(w, path, "agent is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDialogflowCXAgent(project, location, tail[1]))
	return true
}

func handleGCPDialogflowCXValidateAgent(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "agents" {
		return false
	}
	agentID, action, hasAction := strings.Cut(normalizeGCPDialogflowCXActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(agentID) == "" || action != "validate" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":             fmt.Sprintf("projects/%s/locations/%s/agents/%s/validationResult", project, location, agentID),
		"validationErrors": []any{},
	})
	return true
}

func handleGCPDialogflowCXGetAgentValidationResult(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "agents" || strings.TrimSpace(tail[1]) == "" || tail[2] != "validationResult" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":             fmt.Sprintf("projects/%s/locations/%s/agents/%s/validationResult", project, location, tail[1]),
		"validationErrors": []any{},
	})
	return true
}

func handleGCPDialogflowCXExportAgent(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "agents" {
		return false
	}
	agentID, action, hasAction := strings.Cut(normalizeGCPDialogflowCXActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(agentID) == "" || action != "export" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDialogflowCXOperation(project, location, "export-agent-"+agentID))
	return true
}

func handleGCPDialogflowCXRestoreAgent(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "agents" {
		return false
	}
	agentID, action, hasAction := strings.Cut(normalizeGCPDialogflowCXActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(agentID) == "" || action != "restore" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDialogflowCXOperation(project, location, "restore-agent-"+agentID))
	return true
}

func handleGCPDialogflowCXListFlows(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "agents" || strings.TrimSpace(tail[1]) == "" || tail[2] != "flows" {
		return false
	}
	pageSize, start, valid := parseGCPDialogflowCXPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDialogflowCXFlow(project, location, tail[1], "flow-1")}
	return respondGCPDialogflowCXList(w, "flows", items, pageSize, start, path)
}

func handleGCPDialogflowCXGetFlow(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "agents" || strings.TrimSpace(tail[1]) == "" || tail[2] != "flows" || strings.TrimSpace(tail[3]) == "" || strings.Contains(tail[3], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDialogflowCXFlow(project, location, tail[1], tail[3]))
	return true
}

func handleGCPDialogflowCXCreateFlow(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "agents" || strings.TrimSpace(tail[1]) == "" || tail[2] != "flows" {
		return false
	}
	body, valid := decodeGCPDialogflowCXJSONBody(w, r, path)
	if !valid {
		return true
	}
	flow := gcpDialogflowCXBodyMap(body, "flow")
	if len(flow) == 0 {
		respondGCPDialogflowCXInvalidArgument(w, path, "flow is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDialogflowCXFlow(project, location, tail[1], "flow-1"))
	return true
}

func handleGCPDialogflowCXUpdateFlow(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "agents" || strings.TrimSpace(tail[1]) == "" || tail[2] != "flows" || strings.TrimSpace(tail[3]) == "" || strings.Contains(tail[3], ":") {
		return false
	}
	body, valid := decodeGCPDialogflowCXJSONBody(w, r, path)
	if !valid {
		return true
	}
	flow := gcpDialogflowCXBodyMap(body, "flow")
	if len(flow) == 0 {
		respondGCPDialogflowCXInvalidArgument(w, path, "flow is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDialogflowCXFlow(project, location, tail[1], tail[3]))
	return true
}

func handleGCPDialogflowCXValidateFlow(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "agents" || strings.TrimSpace(tail[1]) == "" || tail[2] != "flows" {
		return false
	}
	flowID, action, hasAction := strings.Cut(normalizeGCPDialogflowCXActionSegment(tail[3]), ":")
	if !hasAction || strings.TrimSpace(flowID) == "" || action != "validate" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":             fmt.Sprintf("projects/%s/locations/%s/agents/%s/flows/%s/validationResult", project, location, tail[1], flowID),
		"validationErrors": []any{},
	})
	return true
}

func handleGCPDialogflowCXGetFlowValidationResult(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 5 || tail[0] != "agents" || strings.TrimSpace(tail[1]) == "" || tail[2] != "flows" || strings.TrimSpace(tail[3]) == "" || tail[4] != "validationResult" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":             fmt.Sprintf("projects/%s/locations/%s/agents/%s/flows/%s/validationResult", project, location, tail[1], tail[3]),
		"validationErrors": []any{},
	})
	return true
}

func handleGCPDialogflowCXTrainFlow(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "agents" || strings.TrimSpace(tail[1]) == "" || tail[2] != "flows" {
		return false
	}
	flowID, action, hasAction := strings.Cut(normalizeGCPDialogflowCXActionSegment(tail[3]), ":")
	if !hasAction || strings.TrimSpace(flowID) == "" || action != "train" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDialogflowCXOperation(project, location, "train-flow-"+flowID))
	return true
}

func handleGCPDialogflowCXDetectIntent(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPDialogflowCXSessionAction(w, r, path, "detectIntent")
}

func handleGCPDialogflowCXMatchIntent(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPDialogflowCXSessionAction(w, r, path, "matchIntent")
}

func handleGCPDialogflowCXFulfillIntent(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPDialogflowCXSessionAction(w, r, path, "fulfillIntent")
}

func handleGCPDialogflowCXSessionAction(w http.ResponseWriter, r *http.Request, path string, wantAction string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "agents" || strings.TrimSpace(tail[1]) == "" || tail[2] != "sessions" {
		return false
	}
	sessionID, action, hasAction := strings.Cut(normalizeGCPDialogflowCXActionSegment(tail[3]), ":")
	if !hasAction || strings.TrimSpace(sessionID) == "" || action != wantAction {
		return false
	}
	body, valid := decodeGCPDialogflowCXJSONBody(w, r, path)
	if !valid {
		return true
	}
	switch wantAction {
	case "detectIntent", "matchIntent":
		if _, ok := body["queryInput"]; !ok {
			respondGCPDialogflowCXInvalidArgument(w, path, "queryInput is required")
			return true
		}
	case "fulfillIntent":
		if _, ok := body["matchIntentRequest"]; !ok {
			respondGCPDialogflowCXInvalidArgument(w, path, "matchIntentRequest is required")
			return true
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"responseId": "cx-resp-1",
		"queryResult": map[string]any{
			"text":         "hello from stackyard cx",
			"languageCode": "en",
		},
		"sessionInfo": map[string]any{
			"session": fmt.Sprintf("projects/%s/locations/%s/agents/%s/sessions/%s", project, location, tail[1], sessionID),
		},
		"outputAudio": "",
	})
	return true
}

func handleGCPDialogflowCXListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "operations" {
		return false
	}
	pageSize, start, valid := parseGCPDialogflowCXPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDialogflowCXOperation(project, location, "op-1")}
	return respondGCPDialogflowCXList(w, "operations", items, pageSize, start, path)
}

func handleGCPDialogflowCXGetOperation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDialogflowCXOperation(project, location, tail[1]))
	return true
}

func handleGCPDialogflowCXCancelOperation(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDialogflowCXLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	operationID, action, hasAction := strings.Cut(normalizeGCPDialogflowCXActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(operationID) == "" || action != "cancel" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPDialogflowCXLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v3" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	tail = parts[6:]
	return project, location, tail, len(tail) > 0
}

func parseGCPDialogflowCXPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPDialogflowCXInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPDialogflowCXInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPDialogflowCXList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPDialogflowCXInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPDialogflowCXJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPDialogflowCXInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpDialogflowCXBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func normalizeGCPDialogflowCXActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func gcpDialogflowCXAgent(project, location, agentID string) map[string]any {
	return map[string]any{
		"name":                fmt.Sprintf("projects/%s/locations/%s/agents/%s", project, location, agentID),
		"displayName":         "Stackyard CX Agent",
		"defaultLanguageCode": "en",
		"timeZone":            "America/New_York",
	}
}

func gcpDialogflowCXFlow(project, location, agentID, flowID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/agents/%s/flows/%s", project, location, agentID, flowID),
		"displayName": "Stackyard CX Flow",
		"description": "Flow from Stackyard CX example",
	}
}

func gcpDialogflowCXOperation(project, location, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": true,
	}
}

func respondGCPDialogflowCXInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
