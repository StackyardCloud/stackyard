package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gcpResourceManagerReferenceTime      = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpResourceManagerDisplayNamePattern = regexp.MustCompile(`^[\p{L}\p{N}]([\p{L}\p{N}_\- ]{0,28}[\p{L}\p{N}])?$`)
)

func (s *Server) handleGCPResourceManagerRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_resourcemanager(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPResourceManagerPath(path, hasGCPResourceManagerHint(r)) {
		return false
	}
	if handleGCPResourceManagerV3Router(w, r, path) {
		return true
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPResourceManagerListFolders(w, r, path) {
			return true
		}
		if handleGCPResourceManagerSearchFolders(w, r, path) {
			return true
		}
		if handleGCPResourceManagerGetFolder(w, path) {
			return true
		}
		if handleGCPResourceManagerListOperations(w, r, path) {
			return true
		}
		if handleGCPResourceManagerGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPResourceManagerCreateFolder(w, r, path) {
			return true
		}
		if handleGCPResourceManagerMoveFolder(w, r, path) {
			return true
		}
		if handleGCPResourceManagerUndeleteFolder(w, r, path) {
			return true
		}
		if handleGCPResourceManagerGetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPResourceManagerSetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPResourceManagerTestIAMPermissions(w, r, path) {
			return true
		}
		if handleGCPResourceManagerCancelOperation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPResourceManagerUpdateFolder(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPResourceManagerDeleteFolder(w, r, path) {
			return true
		}
		if handleGCPResourceManagerDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPResourceManagerHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "resourcemanager", "cloudresourcemanager", "resourcemanager_v2", "resourcemanager_v3", "resourcemanager-apiv3":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-resourcemanager-apiv2") ||
		strings.Contains(ua, "stackyard-resourcemanager-apiv3") ||
		strings.Contains(ua, "cloud.google.com/go/resourcemanager")
}

func isGCPResourceManagerPath(path string, includeOperations bool) bool {
	if isGCPResourceManagerV3Path(path) {
		return true
	}
	if isGCPResourceManagerFoldersCollectionPath(path) ||
		isGCPResourceManagerSearchFoldersPath(path) ||
		isGCPResourceManagerFolderPath(path) ||
		isGCPResourceManagerFolderActionPath(path, "move") ||
		isGCPResourceManagerFolderActionPath(path, "undelete") ||
		isGCPResourceManagerFolderActionPath(path, "getIamPolicy") ||
		isGCPResourceManagerFolderActionPath(path, "setIamPolicy") ||
		isGCPResourceManagerFolderActionPath(path, "testIamPermissions") {
		return true
	}
	if !includeOperations {
		return false
	}
	return isGCPResourceManagerOperationsCollectionPath(path) ||
		isGCPResourceManagerOperationPath(path) ||
		isGCPResourceManagerOperationActionPath(path, "cancel")
}

func handleGCPResourceManagerListFolders(w http.ResponseWriter, r *http.Request, path string) bool {
	if !isGCPResourceManagerFoldersCollectionPath(path) {
		return false
	}
	parent := strings.TrimSpace(r.URL.Query().Get("parent"))
	if !isGCPResourceManagerParent(parent) {
		respondGCPResourceManagerInvalidArgument(w, path, "parent must be organizations/{id} or folders/{id}")
		return true
	}
	pageSize, start, valid := parseGCPResourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	showDeleted, err := parseGCPResourceManagerOptionalBool(r.URL.Query().Get("showDeleted"))
	if err != nil {
		respondGCPResourceManagerInvalidArgument(w, path, "showDeleted must be a boolean")
		return true
	}
	items := []map[string]any{
		gcpResourceManagerFolder("1001", parent, "Team Folder", "ACTIVE"),
		gcpResourceManagerFolder("1002", parent, "Archive Folder", "DELETE_REQUESTED"),
	}
	if !showDeleted {
		items = items[:1]
	}
	return respondGCPResourceManagerList(w, "folders", items, pageSize, start, path)
}

func handleGCPResourceManagerSearchFolders(w http.ResponseWriter, r *http.Request, path string) bool {
	if !isGCPResourceManagerSearchFoldersPath(path) {
		return false
	}
	pageSize, start, valid := parseGCPResourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "query is required")
		return true
	}
	items := []map[string]any{
		gcpResourceManagerFolder("1001", "organizations/123456", "Team Folder", "ACTIVE"),
		gcpResourceManagerFolder("1002", "folders/1001", "Archive Folder", "DELETE_REQUESTED"),
	}
	if strings.Contains(strings.ToLower(query), "lifecyclestate=active") {
		items = items[:1]
	}
	if strings.Contains(strings.ToLower(query), "lifecyclestate=delete_requested") {
		items = items[1:]
	}
	if strings.Contains(strings.ToLower(query), "parent=folders/1001") {
		items = items[1:]
	}
	if strings.Contains(strings.ToLower(query), "parent=organizations/123456") {
		items = items[:1]
	}
	return respondGCPResourceManagerList(w, "folders", items, pageSize, start, path)
}

func handleGCPResourceManagerGetFolder(w http.ResponseWriter, path string) bool {
	folderID, ok := parseGCPResourceManagerFolderPath(path)
	if !ok {
		return false
	}
	state := "ACTIVE"
	if strings.Contains(strings.ToLower(folderID), "deleted") {
		state = "DELETE_REQUESTED"
	}
	parent := "organizations/123456"
	if strings.Contains(strings.ToLower(folderID), "child") {
		parent = "folders/1001"
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerFolder(folderID, parent, "Folder "+folderID, state))
	return true
}

func handleGCPResourceManagerCreateFolder(w http.ResponseWriter, r *http.Request, path string) bool {
	if !isGCPResourceManagerFoldersCollectionPath(path) {
		return false
	}
	parent := strings.TrimSpace(r.URL.Query().Get("parent"))
	if !isGCPResourceManagerParent(parent) {
		respondGCPResourceManagerInvalidArgument(w, path, "parent must be organizations/{id} or folders/{id}")
		return true
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	folder := gcpResourceManagerBodyMap(body, "folder")
	if len(folder) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "folder is required")
		return true
	}
	displayName := gcpResourceManagerString(folder, "displayName")
	if !isGCPResourceManagerDisplayName(displayName) {
		respondGCPResourceManagerInvalidArgument(w, path, "folder.displayName is invalid")
		return true
	}
	if folderParent := gcpResourceManagerString(folder, "parent"); folderParent != "" && folderParent != parent {
		respondGCPResourceManagerInvalidArgument(w, path, "folder.parent must match parent query parameter")
		return true
	}
	folderID := "1001"
	if strings.Contains(strings.ToLower(displayName), "archive") {
		folderID = "1002"
	}
	folderName := gcpResourceManagerFolderName(folderID)
	respondJSON(w, http.StatusOK, gcpResourceManagerOperation("create-folder-"+folderID, false, folderName, parent, displayName))
	return true
}

func handleGCPResourceManagerUpdateFolder(w http.ResponseWriter, r *http.Request, path string) bool {
	folderID, ok := parseGCPResourceManagerFolderPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	folder := gcpResourceManagerBodyMap(body, "folder")
	if len(folder) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "folder is required")
		return true
	}
	folderName := gcpResourceManagerString(folder, "name")
	if folderName == "" || folderName != gcpResourceManagerFolderName(folderID) {
		respondGCPResourceManagerInvalidArgument(w, path, "folder.name must match the requested resource")
		return true
	}
	displayName := gcpResourceManagerString(folder, "displayName")
	if !isGCPResourceManagerDisplayName(displayName) {
		respondGCPResourceManagerInvalidArgument(w, path, "folder.displayName is invalid")
		return true
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpResourceManagerString(body, "updateMask"))
	}
	if !isGCPResourceManagerDisplayNameMask(updateMask) {
		respondGCPResourceManagerInvalidArgument(w, path, "updateMask must include display_name")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerFolder(folderID, "organizations/123456", displayName, "ACTIVE"))
	return true
}

func handleGCPResourceManagerMoveFolder(w http.ResponseWriter, r *http.Request, path string) bool {
	folderID, ok := parseGCPResourceManagerFolderActionPath(path, "move")
	if !ok {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	destinationParent := gcpResourceManagerString(body, "destinationParent")
	if !isGCPResourceManagerParent(destinationParent) {
		respondGCPResourceManagerInvalidArgument(w, path, "destinationParent must be organizations/{id} or folders/{id}")
		return true
	}
	if destinationParent == "folders/"+folderID {
		respondGCPResourceManagerFailedPrecondition(w, path, "destinationParent cannot equal folder")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerOperation("move-folder-"+folderID, false, gcpResourceManagerFolderName(folderID), destinationParent, "Moved Folder"))
	return true
}

func handleGCPResourceManagerDeleteFolder(w http.ResponseWriter, r *http.Request, path string) bool {
	folderID, ok := parseGCPResourceManagerFolderPath(path)
	if !ok {
		return false
	}
	if _, err := parseGCPResourceManagerOptionalBool(r.URL.Query().Get("recursiveDelete")); err != nil {
		respondGCPResourceManagerInvalidArgument(w, path, "recursiveDelete must be a boolean")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerFolder(folderID, "organizations/123456", "Folder "+folderID, "DELETE_REQUESTED"))
	return true
}

func handleGCPResourceManagerUndeleteFolder(w http.ResponseWriter, r *http.Request, path string) bool {
	folderID, ok := parseGCPResourceManagerFolderActionPath(path, "undelete")
	if !ok {
		return false
	}
	if _, valid := decodeGCPResourceManagerJSONBody(w, r, path); !valid {
		return true
	}
	if strings.Contains(strings.ToLower(folderID), "active") {
		respondGCPResourceManagerFailedPrecondition(w, path, "folder is not in DELETE_REQUESTED state")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerFolder(folderID, "organizations/123456", "Folder "+folderID, "ACTIVE"))
	return true
}

func handleGCPResourceManagerGetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	folderID, ok := parseGCPResourceManagerFolderActionPath(path, "getIamPolicy")
	if !ok {
		return false
	}
	if _, valid := decodeGCPResourceManagerJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerIAMPolicy(folderID, nil))
	return true
}

func handleGCPResourceManagerSetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	folderID, ok := parseGCPResourceManagerFolderActionPath(path, "setIamPolicy")
	if !ok {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpResourceManagerBodyMap(body, "policy")
	if len(policy) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "policy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerIAMPolicy(folderID, policy))
	return true
}

func handleGCPResourceManagerTestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	_, ok := parseGCPResourceManagerFolderActionPath(path, "testIamPermissions")
	if !ok {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	permissions := gcpResourceManagerStringSlice(body["permissions"])
	if len(permissions) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "permissions is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{"permissions": permissions})
	return true
}

func handleGCPResourceManagerListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	if !isGCPResourceManagerOperationsCollectionPath(path) {
		return false
	}
	pageSize, start, valid := parseGCPResourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpResourceManagerOperation("create-folder-1001", false, "folders/1001", "organizations/123456", "Team Folder"),
		gcpResourceManagerOperation("move-folder-1001", true, "folders/1001", "folders/2000", "Moved Folder"),
	}
	return respondGCPResourceManagerList(w, "operations", items, pageSize, start, path)
}

func handleGCPResourceManagerGetOperation(w http.ResponseWriter, path string) bool {
	opID, ok := parseGCPResourceManagerOperationPath(path)
	if !ok {
		return false
	}
	done := strings.Contains(strings.ToLower(opID), "done") || strings.HasPrefix(strings.ToLower(opID), "move-")
	respondJSON(w, http.StatusOK, gcpResourceManagerOperation(opID, done, "folders/1001", "organizations/123456", "Team Folder"))
	return true
}

func handleGCPResourceManagerCancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	if _, valid := decodeGCPResourceManagerJSONBody(w, r, path); !valid {
		return true
	}
	if _, ok := parseGCPResourceManagerOperationActionPath(path, "cancel"); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPResourceManagerDeleteOperation(w http.ResponseWriter, path string) bool {
	if _, ok := parseGCPResourceManagerOperationPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func isGCPResourceManagerFoldersCollectionPath(path string) bool {
	return strings.TrimSpace(path) == "/gcp/v2/folders"
}

func isGCPResourceManagerSearchFoldersPath(path string) bool {
	return normalizeGCPResourceManagerActionPath(strings.TrimSpace(path)) == "/gcp/v2/folders:search"
}

func isGCPResourceManagerFolderPath(path string) bool {
	_, ok := parseGCPResourceManagerFolderPath(path)
	return ok
}

func isGCPResourceManagerFolderActionPath(path, action string) bool {
	_, ok := parseGCPResourceManagerFolderActionPath(path, action)
	return ok
}

func isGCPResourceManagerOperationsCollectionPath(path string) bool {
	return strings.TrimSpace(path) == "/gcp/v2/operations"
}

func isGCPResourceManagerOperationPath(path string) bool {
	_, ok := parseGCPResourceManagerOperationPath(path)
	return ok
}

func isGCPResourceManagerOperationActionPath(path, action string) bool {
	_, ok := parseGCPResourceManagerOperationActionPath(path, action)
	return ok
}

func parseGCPResourceManagerFolderPath(path string) (folderID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "folders" {
		return "", false
	}
	folderID = strings.TrimSpace(parts[3])
	if folderID == "" || strings.Contains(folderID, ":") {
		return "", false
	}
	return folderID, true
}

func parseGCPResourceManagerFolderActionPath(path, action string) (folderID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "folders" {
		return "", false
	}
	segment := normalizeGCPResourceManagerActionSegment(parts[3])
	folderID, parsedAction, found := strings.Cut(segment, ":")
	if !found || strings.TrimSpace(folderID) == "" || strings.TrimSpace(parsedAction) != action {
		return "", false
	}
	return strings.TrimSpace(folderID), true
}

func parseGCPResourceManagerOperationPath(path string) (operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "operations" {
		return "", false
	}
	operationID = strings.TrimSpace(parts[3])
	if operationID == "" || strings.Contains(operationID, ":") {
		return "", false
	}
	return operationID, true
}

func parseGCPResourceManagerOperationActionPath(path, action string) (operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "operations" {
		return "", false
	}
	segment := normalizeGCPResourceManagerActionSegment(parts[3])
	operationID, parsedAction, found := strings.Cut(segment, ":")
	if !found || strings.TrimSpace(operationID) == "" || strings.TrimSpace(parsedAction) != action {
		return "", false
	}
	return strings.TrimSpace(operationID), true
}

func normalizeGCPResourceManagerActionPath(path string) string {
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func normalizeGCPResourceManagerActionSegment(segment string) string {
	segment = strings.TrimSpace(segment)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func parseGCPResourceManagerPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPResourceManagerInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = strconv.Atoi(token)
		if err != nil || start < 0 {
			respondGCPResourceManagerInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPResourceManagerList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPResourceManagerInvalidArgument(w, path, "pageToken is out of range")
		return false
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

func decodeGCPResourceManagerJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPResourceManagerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpResourceManagerBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpResourceManagerString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func gcpResourceManagerStringSlice(raw any) []string {
	items, _ := raw.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		text, _ := item.(string)
		text = strings.TrimSpace(text)
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func parseGCPResourceManagerOptionalBool(raw string) (bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false, nil
	}
	return strconv.ParseBool(trimmed)
}

func isGCPResourceManagerParent(parent string) bool {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 2 {
		return false
	}
	if parts[0] != "folders" && parts[0] != "organizations" {
		return false
	}
	return strings.TrimSpace(parts[1]) != ""
}

func isGCPResourceManagerDisplayName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	return gcpResourceManagerDisplayNamePattern.MatchString(name)
}

func isGCPResourceManagerDisplayNameMask(mask string) bool {
	mask = strings.TrimSpace(mask)
	if mask == "" {
		return false
	}
	for _, part := range strings.Split(mask, ",") {
		normalized := strings.TrimSpace(part)
		switch normalized {
		case "displayName", "display_name":
		default:
			return false
		}
	}
	return true
}

func gcpResourceManagerFolderName(folderID string) string {
	return "folders/" + strings.TrimSpace(folderID)
}

func gcpResourceManagerFolder(folderID, parent, displayName, state string) map[string]any {
	return map[string]any{
		"name":           gcpResourceManagerFolderName(folderID),
		"parent":         parent,
		"displayName":    displayName,
		"lifecycleState": state,
		"createTime":     gcpResourceManagerReferenceTime.Format(time.RFC3339),
		"updateTime":     gcpResourceManagerReferenceTime.Format(time.RFC3339),
	}
}

func gcpResourceManagerOperation(operationID string, done bool, folderName, destinationParent, displayName string) map[string]any {
	operation := map[string]any{
		"name": "operations/" + operationID,
		"done": done,
		"metadata": map[string]any{
			"@type":             "type.googleapis.com/google.cloud.resourcemanager.v2.FolderOperation",
			"displayName":       displayName,
			"destinationParent": destinationParent,
		},
	}
	if done {
		folderID := strings.TrimPrefix(folderName, "folders/")
		operation["response"] = map[string]any{
			"@type":          "type.googleapis.com/google.cloud.resourcemanager.v2.Folder",
			"name":           folderName,
			"parent":         destinationParent,
			"displayName":    displayName,
			"lifecycleState": "ACTIVE",
			"createTime":     gcpResourceManagerReferenceTime.Format(time.RFC3339),
			"updateTime":     gcpResourceManagerReferenceTime.Format(time.RFC3339),
			"uid":            "folder-" + folderID,
		}
	}
	return operation
}

func gcpResourceManagerIAMPolicy(folderID string, incoming map[string]any) map[string]any {
	bindings := []any{
		map[string]any{
			"role":    "roles/resourcemanager.folderViewer",
			"members": []string{"user:alice@example.com"},
		},
	}
	if incoming != nil {
		if requestedBindings, ok := incoming["bindings"].([]any); ok && len(requestedBindings) > 0 {
			bindings = requestedBindings
		}
	}
	return map[string]any{
		"version":  1,
		"etag":     "cmVzb3VyY2VtYW5hZ2VyLWV0YWc=",
		"bindings": bindings,
		"resource": gcpResourceManagerFolderName(folderID),
	}
}

func respondGCPResourceManagerInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPResourceManagerError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPResourceManagerFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPResourceManagerError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPResourceManagerError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_resourcemanager(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "resourcemanager") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/resourcemanager/sample",
			"service":  "resourcemanager",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
