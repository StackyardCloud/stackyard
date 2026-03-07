package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPCloudBuildRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPCloudBuildPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPCloudBuildListConnections(w, r, path) {
			return true
		}
		if handleGCPCloudBuildGetConnection(w, path) {
			return true
		}
		if handleGCPCloudBuildListRepositories(w, r, path) {
			return true
		}
		if handleGCPCloudBuildGetRepository(w, path) {
			return true
		}
		if handleGCPCloudBuildFetchLinkableRepositories(w, r, path) {
			return true
		}
		if handleGCPCloudBuildFetchGitRefs(w, r, path) {
			return true
		}
		if handleGCPCloudBuildGetIAMPolicy(w, path) {
			return true
		}
		if handleGCPCloudBuildGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPCloudBuildCreateConnection(w, r, path) {
			return true
		}
		if handleGCPCloudBuildCreateRepository(w, r, path) {
			return true
		}
		if handleGCPCloudBuildBatchCreateRepositories(w, r, path) {
			return true
		}
		if handleGCPCloudBuildFetchReadToken(w, path) {
			return true
		}
		if handleGCPCloudBuildFetchReadWriteToken(w, path) {
			return true
		}
		if handleGCPCloudBuildFetchLinkableRepositories(w, r, path) {
			return true
		}
		if handleGCPCloudBuildFetchGitRefs(w, r, path) {
			return true
		}
		if handleGCPCloudBuildGetIAMPolicy(w, path) {
			return true
		}
		if handleGCPCloudBuildSetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPCloudBuildTestIAMPermissions(w, r, path) {
			return true
		}
		if handleGCPCloudBuildCancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPCloudBuildUpdateConnection(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPCloudBuildDeleteConnection(w, path) {
			return true
		}
		if handleGCPCloudBuildDeleteRepository(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPCloudBuildPath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v2/projects/") || !strings.Contains(path, "/locations/") {
		return false
	}
	if _, _, ok := parseGCPCloudBuildConnectionsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPCloudBuildConnectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPCloudBuildRepositoriesCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPCloudBuildRepositoryPath(path); ok {
		return true
	}
	if _, _, _, action, ok := parseGCPCloudBuildRepositoriesActionPath(path); ok {
		if action == "batchCreate" {
			return true
		}
	}
	if _, _, _, action, ok := parseGCPCloudBuildConnectionActionPath(path); ok {
		switch action {
		case "fetchLinkableRepositories", "getIamPolicy", "setIamPolicy", "testIamPermissions":
			return true
		}
	}
	if _, _, _, _, action, _, ok := parseGCPCloudBuildRepositoryActionPath(path); ok {
		switch action {
		case "accessReadToken", "accessReadWriteToken", "fetchGitRefs":
			return true
		}
	}
	if _, _, _, action, ok := parseGCPCloudBuildOperationActionPath(path); ok && action == "cancel" {
		return true
	}
	_, _, _, _, ok := parseGCPCloudBuildOperationPath(path)
	return ok
}

func handleGCPCloudBuildListConnections(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPCloudBuildConnectionsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPCloudBuildPagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpCloudBuildConnection(project, location, "team-connection"),
	}
	return respondGCPCloudBuildList(w, "connections", items, pageSize, start, path)
}

func handleGCPCloudBuildGetConnection(w http.ResponseWriter, path string) bool {
	project, location, connectionID, ok := parseGCPCloudBuildConnectionPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudBuildConnection(project, location, connectionID))
	return true
}

func handleGCPCloudBuildCreateConnection(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPCloudBuildConnectionsCollectionPath(path)
	if !ok {
		return false
	}
	connectionID := strings.TrimSpace(r.URL.Query().Get("connectionId"))
	if connectionID == "" {
		respondGCPCloudBuildInvalidArgument(w, path, "connectionId is required")
		return true
	}

	body, valid := decodeGCPCloudBuildJSONBody(w, r, path)
	if !valid {
		return true
	}
	connection, _ := body["connection"].(map[string]any)
	if len(connection) == 0 {
		connection = body
	}
	if len(connection) == 0 {
		respondGCPCloudBuildInvalidArgument(w, path, "connection is required")
		return true
	}

	respondJSON(w, http.StatusOK, gcpCloudBuildOperation(
		fmt.Sprintf("projects/%s/locations/%s/operations/cloudbuild.createConnection.%s", project, location, connectionID),
	))
	return true
}

func handleGCPCloudBuildUpdateConnection(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, connectionID, ok := parseGCPCloudBuildConnectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPCloudBuildJSONBody(w, r, path)
	if !valid {
		return true
	}
	connection, _ := body["connection"].(map[string]any)
	if len(connection) == 0 {
		connection = body
	}
	name, _ := connection["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPCloudBuildInvalidArgument(w, path, "connection.name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpCloudBuildOperation(
		fmt.Sprintf("projects/%s/locations/%s/operations/cloudbuild.updateConnection.%s", project, location, connectionID),
	))
	return true
}

func handleGCPCloudBuildDeleteConnection(w http.ResponseWriter, path string) bool {
	project, location, connectionID, ok := parseGCPCloudBuildConnectionPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudBuildOperation(
		fmt.Sprintf("projects/%s/locations/%s/operations/cloudbuild.deleteConnection.%s", project, location, connectionID),
	))
	return true
}

func handleGCPCloudBuildListRepositories(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, connectionID, ok := parseGCPCloudBuildRepositoriesCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPCloudBuildPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpCloudBuildRepository(project, location, connectionID, "orders"),
	}
	return respondGCPCloudBuildList(w, "repositories", items, pageSize, start, path)
}

func handleGCPCloudBuildGetRepository(w http.ResponseWriter, path string) bool {
	project, location, connectionID, repositoryID, ok := parseGCPCloudBuildRepositoryPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudBuildRepository(project, location, connectionID, repositoryID))
	return true
}

func handleGCPCloudBuildCreateRepository(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, connectionID, ok := parseGCPCloudBuildRepositoriesCollectionPath(path)
	if !ok {
		return false
	}
	repositoryID := strings.TrimSpace(r.URL.Query().Get("repositoryId"))
	if repositoryID == "" {
		respondGCPCloudBuildInvalidArgument(w, path, "repositoryId is required")
		return true
	}
	body, valid := decodeGCPCloudBuildJSONBody(w, r, path)
	if !valid {
		return true
	}
	repository, _ := body["repository"].(map[string]any)
	if len(repository) == 0 {
		repository = body
	}
	if len(repository) == 0 {
		respondGCPCloudBuildInvalidArgument(w, path, "repository is required")
		return true
	}

	respondJSON(w, http.StatusOK, gcpCloudBuildOperation(
		fmt.Sprintf("projects/%s/locations/%s/operations/cloudbuild.createRepository.%s.%s", project, location, connectionID, repositoryID),
	))
	return true
}

func handleGCPCloudBuildBatchCreateRepositories(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, connectionID, action, ok := parseGCPCloudBuildRepositoriesActionPath(path)
	if !ok || action != "batchCreate" {
		return false
	}
	body, valid := decodeGCPCloudBuildJSONBody(w, r, path)
	if !valid {
		return true
	}
	requests, _ := body["requests"].([]any)
	if len(requests) == 0 {
		respondGCPCloudBuildInvalidArgument(w, path, "requests must contain at least one repository create request")
		return true
	}

	respondJSON(w, http.StatusOK, gcpCloudBuildOperation(
		fmt.Sprintf("projects/%s/locations/%s/operations/cloudbuild.batchCreateRepositories.%s", project, location, connectionID),
	))
	return true
}

func handleGCPCloudBuildDeleteRepository(w http.ResponseWriter, path string) bool {
	project, location, connectionID, repositoryID, ok := parseGCPCloudBuildRepositoryPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudBuildOperation(
		fmt.Sprintf("projects/%s/locations/%s/operations/cloudbuild.deleteRepository.%s.%s", project, location, connectionID, repositoryID),
	))
	return true
}

func handleGCPCloudBuildFetchReadToken(w http.ResponseWriter, path string) bool {
	_, _, _, _, action, _, ok := parseGCPCloudBuildRepositoryActionPath(path)
	if !ok || action != "accessReadToken" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"token":          "stackyard-read-token",
		"expirationTime": "2030-01-01T00:00:00Z",
	})
	return true
}

func handleGCPCloudBuildFetchReadWriteToken(w http.ResponseWriter, path string) bool {
	_, _, _, _, action, _, ok := parseGCPCloudBuildRepositoryActionPath(path)
	if !ok || action != "accessReadWriteToken" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"token":          "stackyard-read-write-token",
		"expirationTime": "2030-01-01T00:00:00Z",
	})
	return true
}

func handleGCPCloudBuildFetchLinkableRepositories(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, connectionID, action, ok := parseGCPCloudBuildConnectionActionPath(path)
	if !ok || action != "fetchLinkableRepositories" {
		return false
	}
	pageSize, start, valid := parseGCPCloudBuildPaginationFromRequest(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpCloudBuildRepository(project, location, connectionID, "orders"),
	}
	return respondGCPCloudBuildList(w, "repositories", items, pageSize, start, path)
}

func handleGCPCloudBuildFetchGitRefs(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, _, action, _, ok := parseGCPCloudBuildRepositoryActionPath(path)
	if !ok || action != "fetchGitRefs" {
		return false
	}

	body, valid := decodeGCPCloudBuildJSONBody(w, r, path)
	if !valid {
		return true
	}
	if refType, ok := body["refType"]; ok {
		switch v := refType.(type) {
		case string:
			if strings.TrimSpace(v) == "" {
				respondGCPCloudBuildInvalidArgument(w, path, "refType must not be empty when provided")
				return true
			}
		case float64:
			if v < 0 {
				respondGCPCloudBuildInvalidArgument(w, path, "refType must be a valid enum value")
				return true
			}
		default:
			respondGCPCloudBuildInvalidArgument(w, path, "refType must be a string or enum number")
			return true
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"refNames": []any{"refs/heads/main", "refs/tags/v1.0.0"},
	})
	return true
}

func handleGCPCloudBuildGetIAMPolicy(w http.ResponseWriter, path string) bool {
	_, _, _, action, ok := parseGCPCloudBuildConnectionActionPath(path)
	if !ok || action != "getIamPolicy" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"version": 1,
		"bindings": []any{
			map[string]any{
				"role": "roles/cloudbuild.connectionViewer",
				"members": []any{
					"user:dev@stackyard.local",
				},
			},
		},
		"etag": "c3RhY2t5YXJkLXBvbGljeS1ldGFn",
	})
	return true
}

func handleGCPCloudBuildSetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, action, ok := parseGCPCloudBuildConnectionActionPath(path)
	if !ok || action != "setIamPolicy" {
		return false
	}
	body, valid := decodeGCPCloudBuildJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy, _ := body["policy"].(map[string]any)
	if len(policy) == 0 {
		policy = body
	}
	if len(policy) == 0 {
		respondGCPCloudBuildInvalidArgument(w, path, "policy is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"version":  1,
		"bindings": []any{},
		"etag":     "c3RhY2t5YXJkLXBvbGljeS1ldGFn",
	})
	return true
}

func handleGCPCloudBuildTestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, action, ok := parseGCPCloudBuildConnectionActionPath(path)
	if !ok || action != "testIamPermissions" {
		return false
	}
	body, valid := decodeGCPCloudBuildJSONBody(w, r, path)
	if !valid {
		return true
	}
	permissions, _ := body["permissions"].([]any)
	if len(permissions) == 0 {
		respondGCPCloudBuildInvalidArgument(w, path, "permissions must contain at least one permission")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"permissions": permissions,
	})
	return true
}

func handleGCPCloudBuildGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, _, ok := parseGCPCloudBuildOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": true,
	})
	return true
}

func handleGCPCloudBuildCancelOperation(w http.ResponseWriter, path string) bool {
	_, _, _, action, ok := parseGCPCloudBuildOperationActionPath(path)
	if !ok || action != "cancel" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPCloudBuildConnectionsCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPCloudBuildConnectionPath(path string) (project, location, connectionID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	connectionID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || connectionID == "" || strings.Contains(connectionID, ":") {
		return "", "", "", false
	}
	return project, location, connectionID, true
}

func parseGCPCloudBuildConnectionActionPath(path string) (project, location, connectionID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	connectionAndAction := normalizeGCPCloudBuildActionSegment(parts[7])
	connectionID, action, found := strings.Cut(connectionAndAction, ":")
	if !found {
		return "", "", "", "", false
	}
	connectionID = strings.TrimSpace(connectionID)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || connectionID == "" || action == "" {
		return "", "", "", "", false
	}
	return project, location, connectionID, action, true
}

func parseGCPCloudBuildRepositoriesCollectionPath(path string) (project, location, connectionID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" || parts[8] != "repositories" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	connectionID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || connectionID == "" {
		return "", "", "", false
	}
	return project, location, connectionID, true
}

func parseGCPCloudBuildRepositoriesActionPath(path string) (project, location, connectionID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	connectionID = strings.TrimSpace(parts[7])
	repositoriesAndAction := normalizeGCPCloudBuildActionSegment(parts[8])
	collection, action, found := strings.Cut(repositoriesAndAction, ":")
	if !found || collection != "repositories" {
		return "", "", "", "", false
	}
	if project == "" || location == "" || connectionID == "" || strings.TrimSpace(action) == "" {
		return "", "", "", "", false
	}
	return project, location, connectionID, strings.TrimSpace(action), true
}

func parseGCPCloudBuildRepositoryPath(path string) (project, location, connectionID, repositoryID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" || parts[8] != "repositories" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	connectionID = strings.TrimSpace(parts[7])
	repositoryID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || connectionID == "" || repositoryID == "" || strings.Contains(repositoryID, ":") {
		return "", "", "", "", false
	}
	return project, location, connectionID, repositoryID, true
}

func parseGCPCloudBuildRepositoryActionPath(path string) (project, location, connectionID, repositoryID, action, fullName string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" || parts[8] != "repositories" {
		return "", "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	connectionID = strings.TrimSpace(parts[7])
	repositoryAndAction := normalizeGCPCloudBuildActionSegment(parts[9])
	repositoryID, action, found := strings.Cut(repositoryAndAction, ":")
	if !found {
		return "", "", "", "", "", "", false
	}
	repositoryID = strings.TrimSpace(repositoryID)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || connectionID == "" || repositoryID == "" || action == "" {
		return "", "", "", "", "", "", false
	}
	fullName = fmt.Sprintf("projects/%s/locations/%s/connections/%s/repositories/%s", project, location, connectionID, repositoryID)
	return project, location, connectionID, repositoryID, action, fullName, true
}

func parseGCPCloudBuildOperationPath(path string) (project, location, operationID, fullName string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	operationID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || operationID == "" || strings.Contains(operationID, ":") {
		return "", "", "", "", false
	}
	fullName = fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID)
	return project, location, operationID, fullName, true
}

func parseGCPCloudBuildOperationActionPath(path string) (project, location, operationID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	opAndAction := normalizeGCPCloudBuildActionSegment(parts[7])
	operationID, action, found := strings.Cut(opAndAction, ":")
	if !found {
		return "", "", "", "", false
	}
	operationID = strings.TrimSpace(operationID)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || operationID == "" || action == "" {
		return "", "", "", "", false
	}
	return project, location, operationID, action, true
}

func parseGCPCloudBuildPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPCloudBuildInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPCloudBuildInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func parseGCPCloudBuildPaginationFromRequest(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	body, valid := decodeGCPCloudBuildJSONBody(w, r, path)
	if !valid {
		return 0, 0, false
	}
	if len(body) == 0 {
		return parseGCPCloudBuildPagination(w, r, path)
	}

	pageSize, err := parseOptionalNonNegativeInt(getStringFromAny(body["pageSize"]))
	if err != nil {
		respondGCPCloudBuildInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start, err = parseOptionalNonNegativeInt(getStringFromAny(body["pageToken"]))
	if err != nil {
		respondGCPCloudBuildInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPCloudBuildList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPCloudBuildInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPCloudBuildJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPCloudBuildInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpCloudBuildConnection(project, location, connectionID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/connections/%s", project, location, connectionID),
		"disabled":    false,
		"reconciling": false,
		"etag":        "stackyard-connection-etag",
	}
}

func gcpCloudBuildRepository(project, location, connectionID, repositoryID string) map[string]any {
	return map[string]any{
		"name":      fmt.Sprintf("projects/%s/locations/%s/connections/%s/repositories/%s", project, location, connectionID, repositoryID),
		"remoteUri": fmt.Sprintf("https://github.com/stackyard/%s.git", repositoryID),
		"etag":      "stackyard-repository-etag",
	}
}

func gcpCloudBuildOperation(name string) map[string]any {
	return map[string]any{
		"name": name,
		"done": true,
	}
}

func normalizeGCPCloudBuildActionSegment(segment string) string {
	normalized := strings.ReplaceAll(segment, "%3A", ":")
	normalized = strings.ReplaceAll(normalized, "%3a", ":")
	return normalized
}

func respondGCPCloudBuildInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
