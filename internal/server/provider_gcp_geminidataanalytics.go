package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) handleGCPGeminiDataAnalyticsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := normalizeGCPGeminiDataAnalyticsPath(rawRequestPath(r))
	if !isGCPGeminiDataAnalyticsPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.cloud.geminidataanalytics.v1beta.DataAgentService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.geminidataanalytics.v1beta.DataChatService/") {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
		if handleGCPGeminiDataAnalyticsLocationDiscovery(w, r, path) {
			return true
		}
		if handleGCPGeminiDataAnalyticsDataAgents(w, r, path) {
			return true
		}
		if handleGCPGeminiDataAnalyticsChat(w, r, path) {
			return true
		}
		if handleGCPGeminiDataAnalyticsConversations(w, r, path) {
			return true
		}
		if handleGCPGeminiDataAnalyticsOperations(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPGeminiDataAnalyticsPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.geminidataanalytics.v1beta.DataAgentService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.geminidataanalytics.v1beta.DataChatService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1beta/projects/") {
		return false
	}
	if isGCPGeminiDataAnalyticsLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/dataAgents") ||
		strings.Contains(path, ":listAccessible") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, "/conversations") ||
		strings.Contains(path, "/messages") ||
		strings.Contains(path, ":chat") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":cancel")
}

func isGCPGeminiDataAnalyticsLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1beta" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func normalizeGCPGeminiDataAnalyticsPath(path string) string {
	normalized := strings.ReplaceAll(path, "%3A", ":")
	normalized = strings.ReplaceAll(normalized, "%3a", ":")
	return normalized
}

func handleGCPGeminiDataAnalyticsLocationDiscovery(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, mode, ok := parseGCPGeminiDataAnalyticsLocationDiscoveryPath(path)
	if !ok {
		return false
	}

	switch mode {
	case "list":
		if r.Method != http.MethodGet {
			return false
		}
		locations := []map[string]any{
			gcpGeminiDataAnalyticsLocation(project, "us-central1"),
			gcpGeminiDataAnalyticsLocation(project, "us-east1"),
		}
		start, end, nextPageToken, ok := gcpGeminiDataAnalyticsPaginate(w, r, path, len(locations))
		if !ok {
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"locations":     locations[start:end],
			"nextPageToken": nextPageToken,
		})
		return true
	case "get":
		if r.Method != http.MethodGet {
			return false
		}
		respondJSON(w, http.StatusOK, gcpGeminiDataAnalyticsLocation(project, location))
		return true
	default:
		return false
	}
}

func handleGCPGeminiDataAnalyticsDataAgents(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, subpath, ok := parseGCPGeminiDataAnalyticsLocationSubpath(path)
	if !ok {
		return false
	}
	parent := gcpGeminiDataAnalyticsParent(project, location)

	switch {
	case subpath == "/dataAgents" && r.Method == http.MethodGet:
		items := []map[string]any{
			gcpGeminiDataAnalyticsDataAgent(parent + "/dataAgents/analytics-agent"),
			gcpGeminiDataAnalyticsDataAgent(parent + "/dataAgents/analytics-agent-2"),
		}
		start, end, nextPageToken, ok := gcpGeminiDataAnalyticsPaginate(w, r, path, len(items))
		if !ok {
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"dataAgents":    items[start:end],
			"nextPageToken": nextPageToken,
		})
		return true
	case subpath == "/dataAgents:listAccessible" && r.Method == http.MethodGet:
		items := []map[string]any{
			gcpGeminiDataAnalyticsDataAgent(parent + "/dataAgents/analytics-agent"),
		}
		start, end, nextPageToken, ok := gcpGeminiDataAnalyticsPaginate(w, r, path, len(items))
		if !ok {
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"dataAgents":    items[start:end],
			"nextPageToken": nextPageToken,
		})
		return true
	case subpath == "/dataAgents" && r.Method == http.MethodPost:
		body, ok := decodeGCPGeminiDataAnalyticsJSONBody(w, r, path)
		if !ok {
			return true
		}
		if _, ok := body["dataAnalyticsAgent"].(map[string]any); !ok {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "dataAnalyticsAgent must be provided",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"name": gcpGeminiDataAnalyticsOperationName(parent, "create-data-agent-op"),
			"done": true,
		})
		return true
	}

	agentID, action, ok := parseGCPGeminiDataAnalyticsDataAgentSubpath(subpath)
	if !ok {
		return false
	}
	agentName := parent + "/dataAgents/" + agentID

	switch {
	case action == "" && r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, gcpGeminiDataAnalyticsDataAgent(agentName))
		return true
	case action == "" && r.Method == http.MethodPatch:
		body, ok := decodeGCPGeminiDataAnalyticsJSONBody(w, r, path)
		if !ok {
			return true
		}
		if name, _ := body["name"].(string); strings.TrimSpace(name) == "" {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "name must be provided in update request body",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"name": gcpGeminiDataAnalyticsOperationName(parent, "update-data-agent-op"),
			"done": true,
		})
		return true
	case action == "" && r.Method == http.MethodDelete:
		respondJSON(w, http.StatusOK, map[string]any{
			"name": gcpGeminiDataAnalyticsOperationName(parent, "delete-data-agent-op"),
			"done": true,
		})
		return true
	case action == "getIamPolicy" && (r.Method == http.MethodGet || r.Method == http.MethodPost):
		if r.Method == http.MethodPost {
			if _, ok := decodeGCPGeminiDataAnalyticsJSONBody(w, r, path); !ok {
				return true
			}
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"bindings": []any{
				map[string]any{
					"role": "roles/viewer",
					"members": []string{
						"user:stackyard@example.com",
					},
				},
			},
			"etag":    "stackyard-etag",
			"version": 1,
		})
		return true
	case action == "setIamPolicy" && r.Method == http.MethodPost:
		body, ok := decodeGCPGeminiDataAnalyticsJSONBody(w, r, path)
		if !ok {
			return true
		}
		policy, ok := body["policy"].(map[string]any)
		if !ok {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "policy must be provided",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		if _, ok := policy["bindings"].([]any); !ok {
			policy["bindings"] = []any{}
		}
		if _, ok := policy["version"]; !ok {
			policy["version"] = 1
		}
		respondJSON(w, http.StatusOK, policy)
		return true
	default:
		return false
	}
}

func handleGCPGeminiDataAnalyticsChat(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, action, ok := parseGCPGeminiDataAnalyticsLocationActionPath(path)
	if !ok || action != "chat" || r.Method != http.MethodPost {
		return false
	}
	parent := gcpGeminiDataAnalyticsParent(project, location)

	body, ok := decodeGCPGeminiDataAnalyticsJSONBody(w, r, path)
	if !ok {
		return true
	}
	if !hasNonEmptyArrayField(body, "messages") {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "messages must be a non-empty array",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	respondJSON(w, http.StatusOK, []any{
		map[string]any{
			"messageId": "msg-1",
			"systemMessage": map[string]any{
				"text": map[string]any{
					"parts": []string{"Stackyard response: revenue is stable month-over-month."},
				},
			},
		},
		map[string]any{
			"messageId": "msg-2",
			"systemMessage": map[string]any{
				"text": map[string]any{
					"parts": []string{"Context used: " + parent + "/dataAgents/analytics-agent"},
				},
			},
		},
	})
	return true
}

func handleGCPGeminiDataAnalyticsConversations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, subpath, ok := parseGCPGeminiDataAnalyticsLocationSubpath(path)
	if !ok {
		return false
	}
	parent := gcpGeminiDataAnalyticsParent(project, location)

	switch {
	case subpath == "/conversations" && r.Method == http.MethodGet:
		items := []map[string]any{
			gcpGeminiDataAnalyticsConversation(parent, "conv-1", "analytics-agent"),
			gcpGeminiDataAnalyticsConversation(parent, "conv-2", "analytics-agent"),
		}
		start, end, nextPageToken, ok := gcpGeminiDataAnalyticsPaginate(w, r, path, len(items))
		if !ok {
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"conversations": items[start:end],
			"nextPageToken": nextPageToken,
		})
		return true
	case subpath == "/conversations" && r.Method == http.MethodPost:
		body, ok := decodeGCPGeminiDataAnalyticsJSONBody(w, r, path)
		if !ok {
			return true
		}
		agents, ok := body["agents"].([]any)
		if !ok || len(agents) == 0 {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "agents must be a non-empty array",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		conversationID := strings.TrimSpace(r.URL.Query().Get("conversationId"))
		if conversationID == "" {
			conversationID = "conv-1"
		}
		agentID := "analytics-agent"
		if agentName, ok := agents[0].(string); ok {
			parts := strings.Split(strings.TrimSpace(agentName), "/")
			if len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) != "" {
				agentID = strings.TrimSpace(parts[len(parts)-1])
			}
		}
		respondJSON(w, http.StatusOK, gcpGeminiDataAnalyticsConversation(parent, conversationID, agentID))
		return true
	}

	conversationID, suffix, ok := parseGCPGeminiDataAnalyticsConversationSubpath(subpath)
	if !ok {
		return false
	}

	switch {
	case suffix == "" && r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, gcpGeminiDataAnalyticsConversation(parent, conversationID, "analytics-agent"))
		return true
	case suffix == "/messages" && r.Method == http.MethodGet:
		items := []map[string]any{
			gcpGeminiDataAnalyticsStoredMessage("msg-1", "show monthly revenue"),
			gcpGeminiDataAnalyticsStoredMessage("msg-2", "summarize by region"),
		}
		start, end, nextPageToken, ok := gcpGeminiDataAnalyticsPaginate(w, r, path, len(items))
		if !ok {
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"messages":      items[start:end],
			"nextPageToken": nextPageToken,
		})
		return true
	default:
		return false
	}
}

func handleGCPGeminiDataAnalyticsOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, subpath, ok := parseGCPGeminiDataAnalyticsLocationSubpath(path)
	if !ok {
		return false
	}
	parent := gcpGeminiDataAnalyticsParent(project, location)

	switch {
	case subpath == "/operations" && r.Method == http.MethodGet:
		items := []map[string]any{
			gcpGeminiDataAnalyticsOperation(gcpGeminiDataAnalyticsOperationName(parent, "op-1")),
			gcpGeminiDataAnalyticsOperation(gcpGeminiDataAnalyticsOperationName(parent, "op-2")),
		}
		start, end, nextPageToken, ok := gcpGeminiDataAnalyticsPaginate(w, r, path, len(items))
		if !ok {
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"operations":    items[start:end],
			"nextPageToken": nextPageToken,
		})
		return true
	}

	operationID, action, ok := parseGCPGeminiDataAnalyticsOperationSubpath(subpath)
	if !ok {
		return false
	}
	operationName := gcpGeminiDataAnalyticsOperationName(parent, operationID)

	switch {
	case action == "" && r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, gcpGeminiDataAnalyticsOperation(operationName))
		return true
	case action == "" && r.Method == http.MethodDelete:
		respondJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
		})
		return true
	case action == "cancel" && r.Method == http.MethodPost:
		if _, ok := decodeGCPGeminiDataAnalyticsJSONBody(w, r, path); !ok {
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"cancelled": true,
			"name":      operationName,
		})
		return true
	default:
		return false
	}
}

func parseGCPGeminiDataAnalyticsLocationDiscoveryPath(path string) (project, location, mode string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	// /gcp/v1beta/projects/{project}/locations
	// /gcp/v1beta/projects/{project}/locations/{location}
	if len(parts) != 5 && len(parts) != 6 {
		return "", "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1beta" || parts[2] != "projects" || parts[4] != "locations" && (len(parts) < 5 || parts[4] != "locations") {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", "", "", false
	}
	if len(parts) == 5 {
		return project, "", "list", true
	}
	location = strings.TrimSpace(parts[5])
	if location == "" {
		return "", "", "", false
	}
	return project, location, "get", true
}

func parseGCPGeminiDataAnalyticsLocationSubpath(path string) (project, location, subpath string, ok bool) {
	const prefix = "/gcp/v1beta/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", "", false
	}
	remainder := strings.TrimPrefix(path, prefix)
	project, remainder, found := strings.Cut(remainder, "/locations/")
	if !found {
		return "", "", "", false
	}
	project = strings.TrimSpace(project)
	if project == "" {
		return "", "", "", false
	}

	locationSegment := strings.TrimSpace(remainder)
	if locationSegment == "" {
		return "", "", "", false
	}
	locationSegment, trailing, found := strings.Cut(locationSegment, "/")
	if found {
		subpath = "/" + strings.TrimSpace(trailing)
	} else {
		subpath = ""
	}
	if strings.Contains(locationSegment, ":") {
		return "", "", "", false
	}
	location = strings.TrimSpace(locationSegment)
	if location == "" {
		return "", "", "", false
	}
	return project, location, subpath, true
}

func parseGCPGeminiDataAnalyticsLocationActionPath(path string) (project, location, action string, ok bool) {
	const prefix = "/gcp/v1beta/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", "", false
	}
	remainder := strings.TrimPrefix(path, prefix)
	project, remainder, found := strings.Cut(remainder, "/locations/")
	if !found {
		return "", "", "", false
	}
	project = strings.TrimSpace(project)
	if project == "" {
		return "", "", "", false
	}
	locationAndAction := strings.TrimSpace(remainder)
	if locationAndAction == "" || strings.Contains(locationAndAction, "/") {
		return "", "", "", false
	}
	location, action, found = strings.Cut(locationAndAction, ":")
	if !found {
		return "", "", "", false
	}
	location = strings.TrimSpace(location)
	action = strings.TrimSpace(action)
	if location == "" || action == "" {
		return "", "", "", false
	}
	return project, location, action, true
}

func parseGCPGeminiDataAnalyticsDataAgentSubpath(subpath string) (agentID, action string, ok bool) {
	if !strings.HasPrefix(subpath, "/dataAgents/") {
		return "", "", false
	}
	rest := strings.TrimPrefix(subpath, "/dataAgents/")
	if rest == "" || strings.Contains(rest, "/") {
		return "", "", false
	}
	agentID = rest
	if strings.Contains(rest, ":") {
		agentID, action, ok = strings.Cut(rest, ":")
		if !ok {
			return "", "", false
		}
		action = strings.TrimSpace(action)
		if action != "getIamPolicy" && action != "setIamPolicy" {
			return "", "", false
		}
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return "", "", false
	}
	return agentID, action, true
}

func parseGCPGeminiDataAnalyticsConversationSubpath(subpath string) (conversationID, suffix string, ok bool) {
	if !strings.HasPrefix(subpath, "/conversations/") {
		return "", "", false
	}
	rest := strings.TrimPrefix(subpath, "/conversations/")
	if rest == "" {
		return "", "", false
	}
	if !strings.Contains(rest, "/") {
		if strings.Contains(rest, ":") {
			return "", "", false
		}
		return strings.TrimSpace(rest), "", strings.TrimSpace(rest) != ""
	}
	conversationID, suffix, _ = strings.Cut(rest, "/")
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" || strings.Contains(conversationID, ":") {
		return "", "", false
	}
	suffix = "/" + strings.TrimSpace(suffix)
	if suffix != "/messages" {
		return "", "", false
	}
	return conversationID, suffix, true
}

func parseGCPGeminiDataAnalyticsOperationSubpath(subpath string) (operationID, action string, ok bool) {
	if !strings.HasPrefix(subpath, "/operations/") {
		return "", "", false
	}
	rest := strings.TrimPrefix(subpath, "/operations/")
	if rest == "" || strings.Contains(rest, "/") {
		return "", "", false
	}
	operationID = rest
	if strings.Contains(rest, ":") {
		operationID, action, ok = strings.Cut(rest, ":")
		if !ok {
			return "", "", false
		}
		action = strings.TrimSpace(action)
		if action != "cancel" {
			return "", "", false
		}
	}
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return "", "", false
	}
	return operationID, action, true
}

func decodeGCPGeminiDataAnalyticsJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return body, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "request body must be valid JSON",
			"provider": providerGCP,
			"path":     path,
		})
		return nil, false
	}
	return body, true
}

func gcpGeminiDataAnalyticsPaginate(w http.ResponseWriter, r *http.Request, path string, total int) (start, end int, nextPageToken string, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return 0, 0, "", false
	}

	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "pageToken must be a non-negative integer offset",
				"provider": providerGCP,
				"path":     path,
			})
			return 0, 0, "", false
		}
	}
	if start > total {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageToken is out of range",
			"provider": providerGCP,
			"path":     path,
		})
		return 0, 0, "", false
	}

	end = total
	if pageSize > 0 && start+pageSize < total {
		end = start + pageSize
	}
	if end < total {
		nextPageToken = strconv.Itoa(end)
	}
	return start, end, nextPageToken, true
}

func gcpGeminiDataAnalyticsParent(project, location string) string {
	return "projects/" + project + "/locations/" + location
}

func gcpGeminiDataAnalyticsDataAgent(name string) map[string]any {
	ts := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	return map[string]any{
		"name":               name,
		"displayName":        "analytics-agent",
		"description":        "Stackyard staged Gemini Data Analytics agent",
		"dataAnalyticsAgent": map[string]any{},
		"createTime":         ts,
		"updateTime":         ts,
	}
}

func gcpGeminiDataAnalyticsConversation(parent, conversationID, agentID string) map[string]any {
	ts := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	return map[string]any{
		"name":         parent + "/conversations/" + conversationID,
		"agents":       []string{parent + "/dataAgents/" + agentID},
		"createTime":   ts,
		"lastUsedTime": ts,
	}
}

func gcpGeminiDataAnalyticsStoredMessage(messageID, text string) map[string]any {
	return map[string]any{
		"messageId": messageID,
		"message": map[string]any{
			"userMessage": map[string]any{
				"text": text,
			},
		},
	}
}

func gcpGeminiDataAnalyticsLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        "projects/" + project + "/locations/" + location,
		"locationId":  location,
		"displayName": strings.ToUpper(location),
		"labels": map[string]any{
			"provider": "stackyard",
		},
	}
}

func gcpGeminiDataAnalyticsOperationName(parent, operationID string) string {
	return parent + "/operations/" + operationID
}

func gcpGeminiDataAnalyticsOperation(name string) map[string]any {
	return map[string]any{
		"name": name,
		"done": true,
	}
}
