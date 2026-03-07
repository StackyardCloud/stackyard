package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPDeveloperConnectRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPDeveloperConnectPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPDeveloperConnectListConnections(w, r, path) {
			return true
		}
		if handleGCPDeveloperConnectGetConnection(w, path) {
			return true
		}
		if handleGCPDeveloperConnectListGitRepositoryLinks(w, r, path) {
			return true
		}
		if handleGCPDeveloperConnectGetGitRepositoryLink(w, path) {
			return true
		}
		if handleGCPDeveloperConnectFetchLinkableGitRepositories(w, r, path) {
			return true
		}
		if handleGCPDeveloperConnectFetchGitHubInstallations(w, path) {
			return true
		}
		if handleGCPDeveloperConnectFetchGitRefs(w, r, path) {
			return true
		}
		if handleGCPDeveloperConnectListOperations(w, r, path) {
			return true
		}
		if handleGCPDeveloperConnectGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPDeveloperConnectCreateConnection(w, r, path) {
			return true
		}
		if handleGCPDeveloperConnectCreateGitRepositoryLink(w, r, path) {
			return true
		}
		if handleGCPDeveloperConnectFetchReadToken(w, path) {
			return true
		}
		if handleGCPDeveloperConnectFetchReadWriteToken(w, path) {
			return true
		}
		if handleGCPDeveloperConnectFetchLinkableGitRepositories(w, r, path) {
			return true
		}
		if handleGCPDeveloperConnectFetchGitHubInstallations(w, path) {
			return true
		}
		if handleGCPDeveloperConnectFetchGitRefs(w, r, path) {
			return true
		}
		if handleGCPDeveloperConnectCancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPDeveloperConnectDeleteConnection(w, path) {
			return true
		}
		if handleGCPDeveloperConnectDeleteGitRepositoryLink(w, path) {
			return true
		}
		if handleGCPDeveloperConnectDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPDeveloperConnectHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	if service == "developerconnect" {
		return true
	}
	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "stackyard-developerconnect-apiv1")
}

func isGCPDeveloperConnectPath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v1/projects/") || !strings.Contains(path, "/locations/") {
		return false
	}
	if _, _, ok := parseGCPDeveloperConnectConnectionsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPDeveloperConnectConnectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPDeveloperConnectConnectionActionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPDeveloperConnectGitRepositoryLinksCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPDeveloperConnectGitRepositoryLinkPath(path); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPDeveloperConnectGitRepositoryLinkActionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPDeveloperConnectOperationsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPDeveloperConnectOperationPath(path); ok {
		return true
	}
	_, _, _, _, ok := parseGCPDeveloperConnectOperationActionPath(path)
	return ok
}

func handleGCPDeveloperConnectListConnections(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPDeveloperConnectConnectionsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPDeveloperConnectPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDeveloperConnectConnection(project, location, "team-connection")}
	return respondGCPDeveloperConnectList(w, "connections", items, pageSize, start, path)
}

func handleGCPDeveloperConnectGetConnection(w http.ResponseWriter, path string) bool {
	project, location, connectionID, ok := parseGCPDeveloperConnectConnectionPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeveloperConnectConnection(project, location, connectionID))
	return true
}

func handleGCPDeveloperConnectCreateConnection(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPDeveloperConnectConnectionsCollectionPath(path)
	if !ok {
		return false
	}
	connectionID := strings.TrimSpace(r.URL.Query().Get("connectionId"))
	if connectionID == "" {
		respondGCPDeveloperConnectInvalidArgument(w, path, "connectionId is required")
		return true
	}
	body, valid := decodeGCPDeveloperConnectJSONBody(w, r, path)
	if !valid {
		return true
	}
	connection, _ := body["connection"].(map[string]any)
	if len(connection) == 0 {
		connection = body
	}
	if len(connection) == 0 {
		respondGCPDeveloperConnectInvalidArgument(w, path, "connection is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDeveloperConnectOperation(project, location, "developerconnect.createConnection."+connectionID))
	return true
}

func handleGCPDeveloperConnectDeleteConnection(w http.ResponseWriter, path string) bool {
	project, location, connectionID, ok := parseGCPDeveloperConnectConnectionPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeveloperConnectOperation(project, location, "developerconnect.deleteConnection."+connectionID))
	return true
}

func handleGCPDeveloperConnectListGitRepositoryLinks(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, connectionID, ok := parseGCPDeveloperConnectGitRepositoryLinksCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPDeveloperConnectPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDeveloperConnectGitRepositoryLink(project, location, connectionID, "orders")}
	return respondGCPDeveloperConnectList(w, "gitRepositoryLinks", items, pageSize, start, path)
}

func handleGCPDeveloperConnectGetGitRepositoryLink(w http.ResponseWriter, path string) bool {
	project, location, connectionID, repoID, ok := parseGCPDeveloperConnectGitRepositoryLinkPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeveloperConnectGitRepositoryLink(project, location, connectionID, repoID))
	return true
}

func handleGCPDeveloperConnectCreateGitRepositoryLink(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, connectionID, ok := parseGCPDeveloperConnectGitRepositoryLinksCollectionPath(path)
	if !ok {
		return false
	}
	repositoryLinkID := strings.TrimSpace(r.URL.Query().Get("gitRepositoryLinkId"))
	if repositoryLinkID == "" {
		respondGCPDeveloperConnectInvalidArgument(w, path, "gitRepositoryLinkId is required")
		return true
	}
	body, valid := decodeGCPDeveloperConnectJSONBody(w, r, path)
	if !valid {
		return true
	}
	link, _ := body["gitRepositoryLink"].(map[string]any)
	if len(link) == 0 {
		link = body
	}
	if len(link) == 0 {
		respondGCPDeveloperConnectInvalidArgument(w, path, "gitRepositoryLink is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDeveloperConnectOperation(project, location, "developerconnect.createGitRepositoryLink."+connectionID+"."+repositoryLinkID))
	return true
}

func handleGCPDeveloperConnectDeleteGitRepositoryLink(w http.ResponseWriter, path string) bool {
	project, location, connectionID, repoID, ok := parseGCPDeveloperConnectGitRepositoryLinkPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeveloperConnectOperation(project, location, "developerconnect.deleteGitRepositoryLink."+connectionID+"."+repoID))
	return true
}

func handleGCPDeveloperConnectFetchReadToken(w http.ResponseWriter, path string) bool {
	_, _, _, _, action, ok := parseGCPDeveloperConnectGitRepositoryLinkActionPath(path)
	if !ok || action != "fetchReadToken" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"token":          "stackyard-read-token",
		"expirationTime": "2030-01-01T00:00:00Z",
		"gitUsername":    "x-access-token",
	})
	return true
}

func handleGCPDeveloperConnectFetchReadWriteToken(w http.ResponseWriter, path string) bool {
	_, _, _, _, action, ok := parseGCPDeveloperConnectGitRepositoryLinkActionPath(path)
	if !ok || action != "fetchReadWriteToken" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"token":          "stackyard-read-write-token",
		"expirationTime": "2030-01-01T00:00:00Z",
		"gitUsername":    "x-access-token",
	})
	return true
}

func handleGCPDeveloperConnectFetchLinkableGitRepositories(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, action, ok := parseGCPDeveloperConnectConnectionActionPath(path)
	if !ok || action != "fetchLinkableGitRepositories" {
		return false
	}
	pageSize, start, valid := parseGCPDeveloperConnectPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{"cloneUri": "https://example.com/stackyard/orders.git"}}
	return respondGCPDeveloperConnectList(w, "linkableGitRepositories", items, pageSize, start, path)
}

func handleGCPDeveloperConnectFetchGitHubInstallations(w http.ResponseWriter, path string) bool {
	_, _, _, action, ok := parseGCPDeveloperConnectConnectionActionPath(path)
	if !ok || action != "fetchGitHubInstallations" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"installations": []any{
			map[string]any{"id": "123", "name": "stackyard-app"},
		},
	})
	return true
}

func handleGCPDeveloperConnectFetchGitRefs(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, _, action, ok := parseGCPDeveloperConnectGitRepositoryLinkActionPath(path)
	if !ok || action != "fetchGitRefs" {
		return false
	}
	pageSize, start, valid := parseGCPDeveloperConnectPagination(w, r, path)
	if !valid {
		return true
	}
	refType := strings.TrimSpace(r.URL.Query().Get("refType"))
	if refType != "" {
		switch refType {
		case "REF_TYPE_UNSPECIFIED", "BRANCH", "TAG", "0", "1", "2":
		default:
			respondGCPDeveloperConnectInvalidArgument(w, path, "refType must be REF_TYPE_UNSPECIFIED, BRANCH, TAG or enum numeric value")
			return true
		}
	}
	items := []string{"refs/heads/main"}
	return respondGCPDeveloperConnectStringList(w, "refNames", items, pageSize, start, path)
}

func handleGCPDeveloperConnectListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPDeveloperConnectOperationsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPDeveloperConnectPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDeveloperConnectOperation(project, location, "op-1")}
	return respondGCPDeveloperConnectList(w, "operations", items, pageSize, start, path)
}

func handleGCPDeveloperConnectGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPDeveloperConnectOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeveloperConnectOperation(project, location, operationID))
	return true
}

func handleGCPDeveloperConnectCancelOperation(w http.ResponseWriter, path string) bool {
	_, _, _, action, ok := parseGCPDeveloperConnectOperationActionPath(path)
	if !ok || action != "cancel" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDeveloperConnectDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPDeveloperConnectOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPDeveloperConnectConnectionsCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPDeveloperConnectConnectionPath(path string) (project, location, connectionID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" {
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

func parseGCPDeveloperConnectConnectionActionPath(path string) (project, location, connectionID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	connectionAction := normalizeGCPDeveloperConnectActionSegment(parts[7])
	connectionID, action, ok = strings.Cut(connectionAction, ":")
	if !ok {
		return "", "", "", "", false
	}
	connectionID = strings.TrimSpace(connectionID)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || connectionID == "" || action == "" {
		return "", "", "", "", false
	}
	return project, location, connectionID, action, true
}

func parseGCPDeveloperConnectGitRepositoryLinksCollectionPath(path string) (project, location, connectionID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" || parts[8] != "gitRepositoryLinks" {
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

func parseGCPDeveloperConnectGitRepositoryLinkPath(path string) (project, location, connectionID, repoID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" || parts[8] != "gitRepositoryLinks" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	connectionID = strings.TrimSpace(parts[7])
	repoID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || connectionID == "" || repoID == "" || strings.Contains(connectionID, ":") || strings.Contains(repoID, ":") {
		return "", "", "", "", false
	}
	return project, location, connectionID, repoID, true
}

func parseGCPDeveloperConnectGitRepositoryLinkActionPath(path string) (project, location, connectionID, repoID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "connections" || parts[8] != "gitRepositoryLinks" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	connectionID = strings.TrimSpace(parts[7])
	repoAction := normalizeGCPDeveloperConnectActionSegment(parts[9])
	repoID, action, ok = strings.Cut(repoAction, ":")
	if !ok {
		return "", "", "", "", "", false
	}
	repoID = strings.TrimSpace(repoID)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || connectionID == "" || repoID == "" || action == "" || strings.Contains(connectionID, ":") {
		return "", "", "", "", "", false
	}
	return project, location, connectionID, repoID, action, true
}

func parseGCPDeveloperConnectOperationsCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPDeveloperConnectOperationPath(path string) (project, location, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	operationID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || operationID == "" || strings.Contains(operationID, ":") {
		return "", "", "", false
	}
	return project, location, operationID, true
}

func parseGCPDeveloperConnectOperationActionPath(path string) (project, location, operationID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	opAction := normalizeGCPDeveloperConnectActionSegment(parts[7])
	operationID, action, ok = strings.Cut(opAction, ":")
	if !ok {
		return "", "", "", "", false
	}
	operationID = strings.TrimSpace(operationID)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || operationID == "" || action == "" {
		return "", "", "", "", false
	}
	return project, location, operationID, action, true
}

func parseGCPDeveloperConnectPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPDeveloperConnectInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPDeveloperConnectInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPDeveloperConnectList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPDeveloperConnectInvalidArgument(w, path, "pageToken is out of range")
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

func respondGCPDeveloperConnectStringList(w http.ResponseWriter, key string, items []string, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPDeveloperConnectInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPDeveloperConnectJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPDeveloperConnectInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func normalizeGCPDeveloperConnectActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func gcpDeveloperConnectConnection(project, location, connectionID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/connections/%s", project, location, connectionID),
		"annotations": map[string]any{"stackyard": "true"},
		"disabled":    false,
		"createTime":  "2026-01-01T00:00:00Z",
		"updateTime":  "2026-01-01T00:00:00Z",
	}
}

func gcpDeveloperConnectGitRepositoryLink(project, location, connectionID, repoID string) map[string]any {
	return map[string]any{
		"name":     fmt.Sprintf("projects/%s/locations/%s/connections/%s/gitRepositoryLinks/%s", project, location, connectionID, repoID),
		"cloneUri": fmt.Sprintf("https://example.com/%s/%s.git", project, repoID),
	}
}

func gcpDeveloperConnectOperation(project, location, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": true,
	}
}

func respondGCPDeveloperConnectInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
