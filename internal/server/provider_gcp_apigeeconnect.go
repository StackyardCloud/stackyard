package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPApigeeConnectRouter(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimSpace(r.URL.Path)
	if path == "" {
		path = rawRequestPath(r)
	}

	if !isGCPApigeeConnectPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.cloud.apigeeconnect.v1.ConnectionService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.apigeeconnect.v1.Tether/") {
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
		if handleGCPApigeeConnectListConnections(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPApigeeConnectEgress(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPApigeeConnectPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.apigeeconnect.v1.ConnectionService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.apigeeconnect.v1.Tether/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/endpoints/") {
		return false
	}
	return strings.Contains(path, "/connections") ||
		strings.Contains(path, ":egress") ||
		strings.Contains(strings.ToLower(path), "%3aegress")
}

func handleGCPApigeeConnectListConnections(w http.ResponseWriter, r *http.Request, path string) bool {
	project, endpoint, ok := parseGCPApigeeConnectConnectionsPath(path)
	if !ok {
		return false
	}

	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPApigeeConnectInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondGCPApigeeConnectInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return true
		}
	}

	connections := []map[string]any{
		gcpApigeeConnectConnection(project, endpoint, "conn-1"),
	}
	if start > len(connections) {
		respondGCPApigeeConnectInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	end := len(connections)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(connections) {
		nextPageToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"connections":   connections[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func handleGCPApigeeConnectEgress(w http.ResponseWriter, r *http.Request, path string) bool {
	project, endpoint, ok := parseGCPApigeeConnectEgressPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPApigeeConnectJSONBody(w, r, path)
	if !valid {
		return true
	}
	id, _ := body["id"].(string)
	if strings.TrimSpace(id) == "" {
		respondGCPApigeeConnectInvalidArgument(w, path, "id is required")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"endpoint": fmt.Sprintf("projects/%s/endpoints/%s", project, endpoint),
		"httpResponse": map[string]any{
			"status": 200,
			"headers": map[string]any{
				"content-type": "application/json",
			},
			"body": "e30=",
		},
	})
	return true
}

func decodeGCPApigeeConnectJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPApigeeConnectInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPApigeeConnectConnectionsPath(path string) (project, endpoint string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	// /gcp/v1/projects/{project}/endpoints/{endpoint}/connections
	if len(parts) != 7 {
		return "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "endpoints" || parts[6] != "connections" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	endpoint = strings.TrimSpace(parts[5])
	if project == "" || endpoint == "" {
		return "", "", false
	}
	return project, endpoint, true
}

func parseGCPApigeeConnectEgressPath(path string) (project, endpoint string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	// /gcp/v1/projects/{project}/endpoints/{endpoint}:egress
	if len(parts) != 6 {
		return "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "endpoints" {
		return "", "", false
	}

	project = strings.TrimSpace(parts[3])
	endpointAction := strings.TrimSpace(parts[5])
	endpointAction = strings.ReplaceAll(endpointAction, "%3A", ":")
	endpointAction = strings.ReplaceAll(endpointAction, "%3a", ":")
	endpoint, action, found := strings.Cut(endpointAction, ":")
	if !found || !strings.EqualFold(action, "egress") {
		return "", "", false
	}
	endpoint = strings.TrimSpace(endpoint)
	if project == "" || endpoint == "" {
		return "", "", false
	}
	return project, endpoint, true
}

func gcpApigeeConnectConnection(project, endpoint, connectionID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/endpoints/%s/connections/%s", project, endpoint, connectionID),
		"endpoint": map[string]any{
			"name": fmt.Sprintf("projects/%s/endpoints/%s", project, endpoint),
		},
		"state": "ESTABLISHED",
	}
}

func respondGCPApigeeConnectInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
