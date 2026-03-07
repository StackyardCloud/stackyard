package server

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var (
	gcpResourceManagerProjectIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{5,29}$`)
	gcpResourceManagerTagShortPattern  = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]{0,61}[A-Za-z0-9])?$`)
)

func handleGCPResourceManagerV3Router(w http.ResponseWriter, r *http.Request, path string) bool {
	if !isGCPResourceManagerV3Path(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPResourceManagerV3ListFolders(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3SearchFolders(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3GetFolder(w, path) {
			return true
		}
		if handleGCPResourceManagerV3ListProjects(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3SearchProjects(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3GetProject(w, path) {
			return true
		}
		if handleGCPResourceManagerV3SearchOrganizations(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3GetOrganization(w, path) {
			return true
		}
		if handleGCPResourceManagerV3ListTagKeys(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3GetTagKey(w, path) {
			return true
		}
		if handleGCPResourceManagerV3GetNamespacedTagKey(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3ListTagValues(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3GetTagValue(w, path) {
			return true
		}
		if handleGCPResourceManagerV3GetNamespacedTagValue(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3ListTagBindings(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3ListEffectiveTags(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3ListTagHolds(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3ListOperations(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3GetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPResourceManagerV3CreateFolder(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3MoveFolder(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3UndeleteFolder(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3CreateProject(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3MoveProject(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3UndeleteProject(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3CreateTagKey(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3CreateTagValue(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3CreateTagBinding(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3CreateTagHold(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3GetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3SetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3TestIAMPermissions(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3CancelOperation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPResourceManagerV3UpdateFolder(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3UpdateProject(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3UpdateTagKey(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3UpdateTagValue(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPResourceManagerV3DeleteFolder(w, r, path) {
			return true
		}
		if handleGCPResourceManagerV3DeleteProject(w, path) {
			return true
		}
		if handleGCPResourceManagerV3DeleteTagKey(w, path) {
			return true
		}
		if handleGCPResourceManagerV3DeleteTagValue(w, path) {
			return true
		}
		if handleGCPResourceManagerV3DeleteTagBinding(w, path) {
			return true
		}
		if handleGCPResourceManagerV3DeleteTagHold(w, path) {
			return true
		}
		if handleGCPResourceManagerV3DeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPResourceManagerV3Path(path string) bool {
	normalized := normalizeGCPResourceManagerActionPath(strings.TrimSpace(path))
	if !strings.HasPrefix(normalized, "/gcp/v3/") {
		return false
	}
	remainder := strings.TrimPrefix(normalized, "/gcp/v3/")
	if remainder == "" {
		return false
	}
	segment := remainder
	if idx := strings.IndexByte(segment, '/'); idx >= 0 {
		segment = segment[:idx]
	}
	if idx := strings.IndexByte(segment, ':'); idx >= 0 {
		segment = segment[:idx]
	}
	switch segment {
	case "folders", "projects", "organizations", "tagKeys", "tagValues", "tagBindings", "effectiveTags", "operations":
		return true
	default:
		return false
	}
}

func handleGCPResourceManagerV3ListFolders(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/folders" {
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
		gcpResourceManagerV3Folder("1001", parent, "Team Folder", "ACTIVE"),
		gcpResourceManagerV3Folder("1002", parent, "Archive Folder", "DELETE_REQUESTED"),
	}
	if !showDeleted {
		items = items[:1]
	}
	return respondGCPResourceManagerList(w, "folders", items, pageSize, start, path)
}

func handleGCPResourceManagerV3SearchFolders(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPResourceManagerActionPath(strings.TrimSpace(path)) != "/gcp/v3/folders:search" {
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
		gcpResourceManagerV3Folder("1001", "organizations/123456", "Team Folder", "ACTIVE"),
		gcpResourceManagerV3Folder("1002", "folders/1001", "Archive Folder", "DELETE_REQUESTED"),
	}
	lower := strings.ToLower(query)
	switch {
	case strings.Contains(lower, "lifecyclestate=active"):
		items = items[:1]
	case strings.Contains(lower, "lifecyclestate=delete_requested"):
		items = items[1:]
	}
	return respondGCPResourceManagerList(w, "folders", items, pageSize, start, path)
}

func handleGCPResourceManagerV3GetFolder(w http.ResponseWriter, path string) bool {
	folderID, ok := parseGCPResourceManagerV3FolderPath(path)
	if !ok {
		return false
	}
	state := "ACTIVE"
	if strings.Contains(strings.ToLower(folderID), "deleted") {
		state = "DELETE_REQUESTED"
	}
	parent := "organizations/123456"
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Folder(folderID, parent, "Folder "+folderID, state))
	return true
}

func handleGCPResourceManagerV3CreateFolder(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/folders" {
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
	displayName := gcpResourceManagerString(folder, "displayName")
	if !isGCPResourceManagerDisplayName(displayName) {
		respondGCPResourceManagerInvalidArgument(w, path, "folder.displayName is invalid")
		return true
	}
	parent := gcpResourceManagerString(folder, "parent")
	if !isGCPResourceManagerParent(parent) {
		respondGCPResourceManagerInvalidArgument(w, path, "folder.parent must be organizations/{id} or folders/{id}")
		return true
	}
	folderID := "1001"
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("create-folder-"+folderID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3UpdateFolder(w http.ResponseWriter, r *http.Request, path string) bool {
	folderID, ok := parseGCPResourceManagerV3FolderPath(path)
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
	if name := gcpResourceManagerString(folder, "name"); name != gcpResourceManagerFolderName(folderID) {
		respondGCPResourceManagerInvalidArgument(w, path, "folder.name must match the requested resource")
		return true
	}
	displayName := gcpResourceManagerString(folder, "displayName")
	if !isGCPResourceManagerDisplayName(displayName) {
		respondGCPResourceManagerInvalidArgument(w, path, "folder.displayName is invalid")
		return true
	}
	updateMask := gcpResourceManagerV3UpdateMask(r, body)
	if !isGCPResourceManagerDisplayNameMask(updateMask) {
		respondGCPResourceManagerInvalidArgument(w, path, "updateMask must include display_name")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("update-folder-"+folderID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3MoveFolder(w http.ResponseWriter, r *http.Request, path string) bool {
	folderID, ok := parseGCPResourceManagerV3FolderActionPath(path, "move")
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
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("move-folder-"+folderID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3DeleteFolder(w http.ResponseWriter, r *http.Request, path string) bool {
	folderID, ok := parseGCPResourceManagerV3FolderPath(path)
	if !ok {
		return false
	}
	if _, err := parseGCPResourceManagerOptionalBool(r.URL.Query().Get("recursiveDelete")); err != nil {
		respondGCPResourceManagerInvalidArgument(w, path, "recursiveDelete must be a boolean")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("delete-folder-"+folderID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3UndeleteFolder(w http.ResponseWriter, r *http.Request, path string) bool {
	folderID, ok := parseGCPResourceManagerV3FolderActionPath(path, "undelete")
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
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("undelete-folder-"+folderID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3ListProjects(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/projects" {
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
		gcpResourceManagerV3Project("415104041262", parent, "stackyard-prod", "Stackyard Prod", "ACTIVE"),
		gcpResourceManagerV3Project("415104041263", parent, "stackyard-archive", "Stackyard Archive", "DELETE_REQUESTED"),
	}
	if !showDeleted {
		items = items[:1]
	}
	return respondGCPResourceManagerList(w, "projects", items, pageSize, start, path)
}

func handleGCPResourceManagerV3SearchProjects(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPResourceManagerActionPath(strings.TrimSpace(path)) != "/gcp/v3/projects:search" {
		return false
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "query is required")
		return true
	}
	pageSize, start, valid := parseGCPResourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpResourceManagerV3Project("415104041262", "organizations/123456", "stackyard-prod", "Stackyard Prod", "ACTIVE"),
		gcpResourceManagerV3Project("415104041263", "folders/1001", "stackyard-archive", "Stackyard Archive", "DELETE_REQUESTED"),
	}
	lower := strings.ToLower(query)
	switch {
	case strings.Contains(lower, "state=active"):
		items = items[:1]
	case strings.Contains(lower, "state=delete_requested"):
		items = items[1:]
	}
	return respondGCPResourceManagerList(w, "projects", items, pageSize, start, path)
}

func handleGCPResourceManagerV3GetProject(w http.ResponseWriter, path string) bool {
	projectID, ok := parseGCPResourceManagerV3ProjectPath(path)
	if !ok {
		return false
	}
	state := "ACTIVE"
	if strings.Contains(strings.ToLower(projectID), "deleted") {
		state = "DELETE_REQUESTED"
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Project(projectID, "organizations/123456", "stackyard-prod", "Project "+projectID, state))
	return true
}

func handleGCPResourceManagerV3CreateProject(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/projects" {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	project := gcpResourceManagerBodyMap(body, "project")
	if len(project) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "project is required")
		return true
	}
	projectID := gcpResourceManagerString(project, "projectId")
	if !isGCPResourceManagerV3ProjectID(projectID) {
		respondGCPResourceManagerInvalidArgument(w, path, "project.projectId is invalid")
		return true
	}
	displayName := gcpResourceManagerString(project, "displayName")
	if !isGCPResourceManagerV3ProjectDisplayName(displayName) {
		respondGCPResourceManagerInvalidArgument(w, path, "project.displayName is invalid")
		return true
	}
	parent := gcpResourceManagerString(project, "parent")
	if parent != "" && !isGCPResourceManagerParent(parent) {
		respondGCPResourceManagerInvalidArgument(w, path, "project.parent must be organizations/{id} or folders/{id}")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("create-project-"+projectID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3UpdateProject(w http.ResponseWriter, r *http.Request, path string) bool {
	projectID, ok := parseGCPResourceManagerV3ProjectPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	project := gcpResourceManagerBodyMap(body, "project")
	if len(project) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "project is required")
		return true
	}
	if name := gcpResourceManagerString(project, "name"); name != gcpResourceManagerV3ProjectName(projectID) {
		respondGCPResourceManagerInvalidArgument(w, path, "project.name must match the requested resource")
		return true
	}
	displayName := gcpResourceManagerString(project, "displayName")
	if !isGCPResourceManagerV3ProjectDisplayName(displayName) {
		respondGCPResourceManagerInvalidArgument(w, path, "project.displayName is invalid")
		return true
	}
	updateMask := gcpResourceManagerV3UpdateMask(r, body)
	if !isGCPResourceManagerV3ProjectMask(updateMask) {
		respondGCPResourceManagerInvalidArgument(w, path, "updateMask must include display_name or labels")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("update-project-"+projectID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3MoveProject(w http.ResponseWriter, r *http.Request, path string) bool {
	projectID, ok := parseGCPResourceManagerV3ProjectActionPath(path, "move")
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
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("move-project-"+projectID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3DeleteProject(w http.ResponseWriter, path string) bool {
	projectID, ok := parseGCPResourceManagerV3ProjectPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("delete-project-"+projectID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3UndeleteProject(w http.ResponseWriter, r *http.Request, path string) bool {
	projectID, ok := parseGCPResourceManagerV3ProjectActionPath(path, "undelete")
	if !ok {
		return false
	}
	if _, valid := decodeGCPResourceManagerJSONBody(w, r, path); !valid {
		return true
	}
	if strings.Contains(strings.ToLower(projectID), "active") {
		respondGCPResourceManagerFailedPrecondition(w, path, "project is not in DELETE_REQUESTED state")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("undelete-project-"+projectID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3SearchOrganizations(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPResourceManagerActionPath(strings.TrimSpace(path)) != "/gcp/v3/organizations:search" {
		return false
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "query is required")
		return true
	}
	pageSize, start, valid := parseGCPResourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpResourceManagerV3Organization("123456", "example.com", "ACTIVE"),
		gcpResourceManagerV3Organization("123457", "example.org", "ACTIVE"),
	}
	lower := strings.ToLower(query)
	if strings.Contains(lower, "domain:example.com") {
		items = items[:1]
	}
	return respondGCPResourceManagerList(w, "organizations", items, pageSize, start, path)
}

func handleGCPResourceManagerV3GetOrganization(w http.ResponseWriter, path string) bool {
	orgID, ok := parseGCPResourceManagerV3OrganizationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Organization(orgID, "example.com", "ACTIVE"))
	return true
}

func handleGCPResourceManagerV3ListTagKeys(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/tagKeys" {
		return false
	}
	parent := strings.TrimSpace(r.URL.Query().Get("parent"))
	if !isGCPResourceManagerV3TagKeyParent(parent) {
		respondGCPResourceManagerInvalidArgument(w, path, "parent must be organizations/{id} or projects/{id}")
		return true
	}
	pageSize, start, valid := parseGCPResourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpResourceManagerV3TagKey("2001", parent, "env"),
		gcpResourceManagerV3TagKey("2002", parent, "owner"),
	}
	return respondGCPResourceManagerList(w, "tagKeys", items, pageSize, start, path)
}

func handleGCPResourceManagerV3GetTagKey(w http.ResponseWriter, path string) bool {
	tagKeyID, ok := parseGCPResourceManagerV3TagKeyPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3TagKey(tagKeyID, "organizations/123456", "env"))
	return true
}

func handleGCPResourceManagerV3GetNamespacedTagKey(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/tagKeys/namespaced" {
		return false
	}
	namespaced := strings.TrimSpace(r.URL.Query().Get("name"))
	if namespaced == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "name is required")
		return true
	}
	shortName := "env"
	if idx := strings.LastIndex(namespaced, "/"); idx >= 0 && idx+1 < len(namespaced) {
		shortName = namespaced[idx+1:]
	}
	item := gcpResourceManagerV3TagKey("2001", "organizations/123456", shortName)
	item["namespacedName"] = namespaced
	respondJSON(w, http.StatusOK, item)
	return true
}

func handleGCPResourceManagerV3CreateTagKey(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/tagKeys" {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	tagKey := gcpResourceManagerBodyMap(body, "tagKey")
	if len(tagKey) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "tagKey is required")
		return true
	}
	parent := gcpResourceManagerString(tagKey, "parent")
	if !isGCPResourceManagerV3TagKeyParent(parent) {
		respondGCPResourceManagerInvalidArgument(w, path, "tagKey.parent must be organizations/{id} or projects/{id}")
		return true
	}
	shortName := gcpResourceManagerString(tagKey, "shortName")
	if !isGCPResourceManagerV3TagShortName(shortName) {
		respondGCPResourceManagerInvalidArgument(w, path, "tagKey.shortName is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("create-tagkey-2001", false, nil, ""))
	return true
}

func handleGCPResourceManagerV3UpdateTagKey(w http.ResponseWriter, r *http.Request, path string) bool {
	tagKeyID, ok := parseGCPResourceManagerV3TagKeyPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	tagKey := gcpResourceManagerBodyMap(body, "tagKey")
	if len(tagKey) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "tagKey is required")
		return true
	}
	if name := gcpResourceManagerString(tagKey, "name"); name != gcpResourceManagerV3TagKeyName(tagKeyID) {
		respondGCPResourceManagerInvalidArgument(w, path, "tagKey.name must match the requested resource")
		return true
	}
	if desc := gcpResourceManagerString(tagKey, "description"); desc == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "tagKey.description is required")
		return true
	}
	updateMask := gcpResourceManagerV3UpdateMask(r, body)
	if !isGCPResourceManagerV3TagMask(updateMask) {
		respondGCPResourceManagerInvalidArgument(w, path, "updateMask must include description or etag")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("update-tagkey-"+tagKeyID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3DeleteTagKey(w http.ResponseWriter, path string) bool {
	tagKeyID, ok := parseGCPResourceManagerV3TagKeyPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("delete-tagkey-"+tagKeyID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3ListTagValues(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/tagValues" {
		return false
	}
	parent := strings.TrimSpace(r.URL.Query().Get("parent"))
	if !isGCPResourceManagerV3TagValueParent(parent) {
		respondGCPResourceManagerInvalidArgument(w, path, "parent must be tagKeys/{id}")
		return true
	}
	pageSize, start, valid := parseGCPResourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpResourceManagerV3TagValue("3001", parent, "prod"),
		gcpResourceManagerV3TagValue("3002", parent, "dev"),
	}
	return respondGCPResourceManagerList(w, "tagValues", items, pageSize, start, path)
}

func handleGCPResourceManagerV3GetTagValue(w http.ResponseWriter, path string) bool {
	tagValueID, ok := parseGCPResourceManagerV3TagValuePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3TagValue(tagValueID, "tagKeys/2001", "prod"))
	return true
}

func handleGCPResourceManagerV3GetNamespacedTagValue(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/tagValues/namespaced" {
		return false
	}
	namespaced := strings.TrimSpace(r.URL.Query().Get("name"))
	if namespaced == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "name is required")
		return true
	}
	shortName := "prod"
	if idx := strings.LastIndex(namespaced, "/"); idx >= 0 && idx+1 < len(namespaced) {
		shortName = namespaced[idx+1:]
	}
	item := gcpResourceManagerV3TagValue("3001", "tagKeys/2001", shortName)
	item["namespacedName"] = namespaced
	respondJSON(w, http.StatusOK, item)
	return true
}

func handleGCPResourceManagerV3CreateTagValue(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/tagValues" {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	tagValue := gcpResourceManagerBodyMap(body, "tagValue")
	if len(tagValue) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "tagValue is required")
		return true
	}
	parent := gcpResourceManagerString(tagValue, "parent")
	if !isGCPResourceManagerV3TagValueParent(parent) {
		respondGCPResourceManagerInvalidArgument(w, path, "tagValue.parent must be tagKeys/{id}")
		return true
	}
	shortName := gcpResourceManagerString(tagValue, "shortName")
	if !isGCPResourceManagerV3TagShortName(shortName) {
		respondGCPResourceManagerInvalidArgument(w, path, "tagValue.shortName is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("create-tagvalue-3001", false, nil, ""))
	return true
}

func handleGCPResourceManagerV3UpdateTagValue(w http.ResponseWriter, r *http.Request, path string) bool {
	tagValueID, ok := parseGCPResourceManagerV3TagValuePath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	tagValue := gcpResourceManagerBodyMap(body, "tagValue")
	if len(tagValue) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "tagValue is required")
		return true
	}
	if name := gcpResourceManagerString(tagValue, "name"); name != gcpResourceManagerV3TagValueName(tagValueID) {
		respondGCPResourceManagerInvalidArgument(w, path, "tagValue.name must match the requested resource")
		return true
	}
	if desc := gcpResourceManagerString(tagValue, "description"); desc == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "tagValue.description is required")
		return true
	}
	updateMask := gcpResourceManagerV3UpdateMask(r, body)
	if !isGCPResourceManagerV3TagMask(updateMask) {
		respondGCPResourceManagerInvalidArgument(w, path, "updateMask must include description or etag")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("update-tagvalue-"+tagValueID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3DeleteTagValue(w http.ResponseWriter, path string) bool {
	tagValueID, ok := parseGCPResourceManagerV3TagValuePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("delete-tagvalue-"+tagValueID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3ListTagBindings(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/tagBindings" {
		return false
	}
	parent := strings.TrimSpace(r.URL.Query().Get("parent"))
	if parent == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, start, valid := parseGCPResourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpResourceManagerV3TagBinding(parent, "tagValues/3001"),
	}
	return respondGCPResourceManagerList(w, "tagBindings", items, pageSize, start, path)
}

func handleGCPResourceManagerV3CreateTagBinding(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/tagBindings" {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	tagBinding := gcpResourceManagerBodyMap(body, "tagBinding")
	if len(tagBinding) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "tagBinding is required")
		return true
	}
	parent := gcpResourceManagerString(tagBinding, "parent")
	tagValue := gcpResourceManagerString(tagBinding, "tagValue")
	tagValueNamespaced := gcpResourceManagerString(tagBinding, "tagValueNamespacedName")
	if parent == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "tagBinding.parent is required")
		return true
	}
	if tagValue == "" && tagValueNamespaced == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "tagBinding.tagValue or tagBinding.tagValueNamespacedName is required")
		return true
	}
	if tagValue != "" && tagValueNamespaced != "" {
		respondGCPResourceManagerInvalidArgument(w, path, "tagBinding cannot include both tagValue and tagValueNamespacedName")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("create-tagbinding-3001", false, nil, ""))
	return true
}

func handleGCPResourceManagerV3DeleteTagBinding(w http.ResponseWriter, path string) bool {
	tagBindingName, ok := parseGCPResourceManagerV3TagBindingPath(path)
	if !ok {
		return false
	}
	opID := "delete-tagbinding-" + strings.ReplaceAll(strings.TrimPrefix(tagBindingName, "tagBindings/"), "/", "-")
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation(opID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3ListEffectiveTags(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/effectiveTags" {
		return false
	}
	parent := strings.TrimSpace(r.URL.Query().Get("parent"))
	if parent == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, start, valid := parseGCPResourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpResourceManagerV3EffectiveTag(parent, "tagKeys/2001", "tagValues/3001"),
	}
	return respondGCPResourceManagerList(w, "effectiveTags", items, pageSize, start, path)
}

func handleGCPResourceManagerV3ListTagHolds(w http.ResponseWriter, r *http.Request, path string) bool {
	tagValueID, ok := parseGCPResourceManagerV3TagHoldsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPResourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpResourceManagerV3TagHold(tagValueID, "hold-1"),
	}
	return respondGCPResourceManagerList(w, "tagHolds", items, pageSize, start, path)
}

func handleGCPResourceManagerV3CreateTagHold(w http.ResponseWriter, r *http.Request, path string) bool {
	tagValueID, ok := parseGCPResourceManagerV3TagHoldsCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPResourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	tagHold := gcpResourceManagerBodyMap(body, "tagHold")
	if len(tagHold) == 0 {
		respondGCPResourceManagerInvalidArgument(w, path, "tagHold is required")
		return true
	}
	if holder := gcpResourceManagerString(tagHold, "holder"); holder == "" {
		respondGCPResourceManagerInvalidArgument(w, path, "tagHold.holder is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("create-taghold-"+tagValueID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3DeleteTagHold(w http.ResponseWriter, path string) bool {
	tagValueID, holdID, ok := parseGCPResourceManagerV3TagHoldPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation("delete-taghold-"+tagValueID+"-"+holdID, false, nil, ""))
	return true
}

func handleGCPResourceManagerV3GetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, ok := parseGCPResourceManagerV3IAMResourceActionPath(path, "getIamPolicy")
	if !ok {
		return false
	}
	if _, valid := decodeGCPResourceManagerJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, gcpResourceManagerV3IAMPolicy(resource, nil))
	return true
}

func handleGCPResourceManagerV3SetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, ok := parseGCPResourceManagerV3IAMResourceActionPath(path, "setIamPolicy")
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
	respondJSON(w, http.StatusOK, gcpResourceManagerV3IAMPolicy(resource, policy))
	return true
}

func handleGCPResourceManagerV3TestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	_, ok := parseGCPResourceManagerV3IAMResourceActionPath(path, "testIamPermissions")
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

func handleGCPResourceManagerV3ListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(path) != "/gcp/v3/operations" {
		return false
	}
	pageSize, start, valid := parseGCPResourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpResourceManagerV3Operation("create-folder-1001", false, nil, ""),
		gcpResourceManagerV3Operation("move-project-415104041262", true, nil, ""),
	}
	return respondGCPResourceManagerList(w, "operations", items, pageSize, start, path)
}

func handleGCPResourceManagerV3GetOperation(w http.ResponseWriter, path string) bool {
	opID, ok := parseGCPResourceManagerV3OperationPath(path)
	if !ok {
		return false
	}
	done := strings.Contains(strings.ToLower(opID), "done") || strings.HasPrefix(strings.ToLower(opID), "move-")
	respondJSON(w, http.StatusOK, gcpResourceManagerV3Operation(opID, done, nil, ""))
	return true
}

func handleGCPResourceManagerV3CancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	if _, valid := decodeGCPResourceManagerJSONBody(w, r, path); !valid {
		return true
	}
	if _, ok := parseGCPResourceManagerV3OperationActionPath(path, "cancel"); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPResourceManagerV3DeleteOperation(w http.ResponseWriter, path string) bool {
	if _, ok := parseGCPResourceManagerV3OperationPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPResourceManagerV3FolderPath(path string) (string, bool) {
	return parseGCPResourceManagerV3SimpleResourcePath(path, "folders")
}

func parseGCPResourceManagerV3ProjectPath(path string) (string, bool) {
	return parseGCPResourceManagerV3SimpleResourcePath(path, "projects")
}

func parseGCPResourceManagerV3OrganizationPath(path string) (string, bool) {
	return parseGCPResourceManagerV3SimpleResourcePath(path, "organizations")
}

func parseGCPResourceManagerV3TagKeyPath(path string) (string, bool) {
	id, ok := parseGCPResourceManagerV3SimpleResourcePath(path, "tagKeys")
	if !ok || strings.EqualFold(id, "namespaced") {
		return "", false
	}
	return id, true
}

func parseGCPResourceManagerV3TagValuePath(path string) (string, bool) {
	id, ok := parseGCPResourceManagerV3SimpleResourcePath(path, "tagValues")
	if !ok || strings.EqualFold(id, "namespaced") {
		return "", false
	}
	return id, true
}

func parseGCPResourceManagerV3SimpleResourcePath(path, resource string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v3" || parts[2] != resource {
		return "", false
	}
	id := strings.TrimSpace(parts[3])
	if id == "" || strings.Contains(id, ":") {
		return "", false
	}
	return id, true
}

func parseGCPResourceManagerV3FolderActionPath(path, action string) (string, bool) {
	return parseGCPResourceManagerV3SimpleActionPath(path, "folders", action)
}

func parseGCPResourceManagerV3ProjectActionPath(path, action string) (string, bool) {
	return parseGCPResourceManagerV3SimpleActionPath(path, "projects", action)
}

func parseGCPResourceManagerV3SimpleActionPath(path, resource, action string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v3" || parts[2] != resource {
		return "", false
	}
	segment := normalizeGCPResourceManagerActionSegment(parts[3])
	id, parsedAction, found := strings.Cut(segment, ":")
	if !found || strings.TrimSpace(id) == "" || strings.TrimSpace(parsedAction) != action {
		return "", false
	}
	return strings.TrimSpace(id), true
}

func parseGCPResourceManagerV3TagBindingPath(path string) (string, bool) {
	trimmed := strings.TrimSpace(path)
	if !strings.HasPrefix(trimmed, "/gcp/v3/tagBindings/") {
		return "", false
	}
	name := strings.TrimPrefix(strings.Trim(trimmed, "/"), "gcp/v3/")
	if strings.TrimSpace(name) == "" || strings.Contains(strings.ToLower(name), ":") {
		return "", false
	}
	return name, true
}

func parseGCPResourceManagerV3TagHoldsCollectionPath(path string) (tagValueID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v3" || parts[2] != "tagValues" || parts[4] != "tagHolds" {
		return "", false
	}
	tagValueID = strings.TrimSpace(parts[3])
	if tagValueID == "" {
		return "", false
	}
	return tagValueID, true
}

func parseGCPResourceManagerV3TagHoldPath(path string) (tagValueID, holdID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v3" || parts[2] != "tagValues" || parts[4] != "tagHolds" {
		return "", "", false
	}
	tagValueID = strings.TrimSpace(parts[3])
	holdID = strings.TrimSpace(parts[5])
	if tagValueID == "" || holdID == "" || strings.Contains(holdID, ":") {
		return "", "", false
	}
	return tagValueID, holdID, true
}

func parseGCPResourceManagerV3IAMResourceActionPath(path, action string) (resource string, ok bool) {
	normalized := normalizeGCPResourceManagerActionPath(strings.TrimSpace(path))
	if !strings.HasPrefix(normalized, "/gcp/v3/") {
		return "", false
	}
	remainder := strings.TrimPrefix(normalized, "/gcp/v3/")
	resource, parsedAction, found := strings.Cut(remainder, ":")
	if !found || strings.TrimSpace(resource) == "" || strings.TrimSpace(parsedAction) != action {
		return "", false
	}
	if !isGCPResourceManagerV3IAMResource(resource) {
		return "", false
	}
	return resource, true
}

func isGCPResourceManagerV3IAMResource(resource string) bool {
	prefixes := []string{"folders/", "projects/", "organizations/", "tagKeys/", "tagValues/"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(resource, prefix) && strings.TrimSpace(strings.TrimPrefix(resource, prefix)) != "" {
			return true
		}
	}
	return false
}

func parseGCPResourceManagerV3OperationPath(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v3" || parts[2] != "operations" {
		return "", false
	}
	opID := strings.TrimSpace(parts[3])
	if opID == "" || strings.Contains(opID, ":") {
		return "", false
	}
	return opID, true
}

func parseGCPResourceManagerV3OperationActionPath(path, action string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v3" || parts[2] != "operations" {
		return "", false
	}
	segment := normalizeGCPResourceManagerActionSegment(parts[3])
	opID, parsedAction, found := strings.Cut(segment, ":")
	if !found || strings.TrimSpace(opID) == "" || strings.TrimSpace(parsedAction) != action {
		return "", false
	}
	return strings.TrimSpace(opID), true
}

func isGCPResourceManagerV3ProjectID(projectID string) bool {
	if !gcpResourceManagerProjectIDPattern.MatchString(projectID) {
		return false
	}
	return !strings.HasSuffix(projectID, "-")
}

func isGCPResourceManagerV3ProjectDisplayName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 4 || len(name) > 30 {
		return false
	}
	return true
}

func isGCPResourceManagerV3TagShortName(name string) bool {
	return gcpResourceManagerTagShortPattern.MatchString(strings.TrimSpace(name))
}

func isGCPResourceManagerV3TagKeyParent(parent string) bool {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 2 {
		return false
	}
	if parts[0] != "organizations" && parts[0] != "projects" {
		return false
	}
	return strings.TrimSpace(parts[1]) != ""
}

func isGCPResourceManagerV3TagValueParent(parent string) bool {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	return len(parts) == 2 && parts[0] == "tagKeys" && strings.TrimSpace(parts[1]) != ""
}

func gcpResourceManagerV3UpdateMask(r *http.Request, body map[string]any) string {
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask != "" {
		return updateMask
	}
	if value := gcpResourceManagerString(body, "updateMask"); value != "" {
		return value
	}
	if raw, ok := body["updateMask"].(map[string]any); ok {
		if paths, ok := raw["paths"].([]any); ok {
			parts := make([]string, 0, len(paths))
			for _, path := range paths {
				if text, ok := path.(string); ok && strings.TrimSpace(text) != "" {
					parts = append(parts, strings.TrimSpace(text))
				}
			}
			return strings.Join(parts, ",")
		}
	}
	return ""
}

func isGCPResourceManagerV3ProjectMask(mask string) bool {
	mask = strings.TrimSpace(mask)
	if mask == "" {
		return false
	}
	for _, part := range strings.Split(mask, ",") {
		normalized := strings.TrimSpace(part)
		switch normalized {
		case "display_name", "displayName", "labels":
		default:
			return false
		}
	}
	return true
}

func isGCPResourceManagerV3TagMask(mask string) bool {
	mask = strings.TrimSpace(mask)
	if mask == "" {
		return false
	}
	for _, part := range strings.Split(mask, ",") {
		normalized := strings.TrimSpace(part)
		switch normalized {
		case "description", "etag":
		default:
			return false
		}
	}
	return true
}

func gcpResourceManagerV3ProjectName(projectID string) string {
	return "projects/" + strings.TrimSpace(projectID)
}

func gcpResourceManagerV3TagKeyName(tagKeyID string) string {
	return "tagKeys/" + strings.TrimSpace(tagKeyID)
}

func gcpResourceManagerV3TagValueName(tagValueID string) string {
	return "tagValues/" + strings.TrimSpace(tagValueID)
}

func gcpResourceManagerV3Folder(folderID, parent, displayName, state string) map[string]any {
	return map[string]any{
		"name":           gcpResourceManagerFolderName(folderID),
		"parent":         parent,
		"displayName":    displayName,
		"lifecycleState": state,
		"createTime":     gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
		"updateTime":     gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
	}
}

func gcpResourceManagerV3Project(projectNumber, parent, projectID, displayName, state string) map[string]any {
	return map[string]any{
		"name":        gcpResourceManagerV3ProjectName(projectNumber),
		"parent":      parent,
		"projectId":   projectID,
		"displayName": displayName,
		"state":       state,
		"createTime":  gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
		"updateTime":  gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
		"etag":        "cmVzb3VyY2VtYW5hZ2VyLXByb2plY3QtZXRhZw==",
		"labels": map[string]string{
			"environment": "test",
			"owner":       "stackyard",
		},
	}
}

func gcpResourceManagerV3Organization(orgID, displayName, state string) map[string]any {
	return map[string]any{
		"name":                "organizations/" + strings.TrimSpace(orgID),
		"displayName":         displayName,
		"state":               state,
		"directoryCustomerId": "C0123abc",
		"createTime":          gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
		"updateTime":          gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
		"etag":                "cmVzb3VyY2VtYW5hZ2VyLW9yZy1ldGFn",
	}
}

func gcpResourceManagerV3TagKey(tagKeyID, parent, shortName string) map[string]any {
	return map[string]any{
		"name":           gcpResourceManagerV3TagKeyName(tagKeyID),
		"parent":         parent,
		"shortName":      shortName,
		"namespacedName": "123456/" + shortName,
		"description":    "Tag key for " + shortName,
		"createTime":     gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
		"updateTime":     gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
		"etag":           "cmVzb3VyY2VtYW5hZ2VyLXRhZ2tleS1ldGFn",
		"purpose":        "GCE_FIREWALL",
		"purposeData": map[string]string{
			"network": "default",
		},
	}
}

func gcpResourceManagerV3TagValue(tagValueID, parent, shortName string) map[string]any {
	return map[string]any{
		"name":           gcpResourceManagerV3TagValueName(tagValueID),
		"parent":         parent,
		"shortName":      shortName,
		"namespacedName": "123456/env/" + shortName,
		"description":    "Tag value for " + shortName,
		"createTime":     gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
		"updateTime":     gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
		"etag":           "cmVzb3VyY2VtYW5hZ2VyLXRhZ3ZhbHVlLWV0YWc=",
	}
}

func gcpResourceManagerV3TagBinding(parent, tagValue string) map[string]any {
	encodedParent := strings.NewReplacer("/", "%2F", ":", "%3A").Replace(parent)
	return map[string]any{
		"name":                   "tagBindings/" + encodedParent + "/" + tagValue,
		"parent":                 parent,
		"tagValue":               tagValue,
		"tagValueNamespacedName": "123456/env/prod",
	}
}

func gcpResourceManagerV3EffectiveTag(parent, tagKey, tagValue string) map[string]any {
	return map[string]any{
		"tagValue":           tagValue,
		"namespacedTagValue": "123456/env/prod",
		"tagKey":             tagKey,
		"namespacedTagKey":   "123456/env",
		"tagKeyParentName":   "organizations/123456",
		"inherited":          !strings.Contains(parent, "projects/"),
	}
}

func gcpResourceManagerV3TagHold(tagValueID, holdID string) map[string]any {
	return map[string]any{
		"name":       fmt.Sprintf("tagValues/%s/tagHolds/%s", tagValueID, holdID),
		"holder":     "//cloudresourcemanager.googleapis.com/projects/415104041262",
		"origin":     "stackyard",
		"helpLink":   "https://cloud.google.com/resource-manager/docs/tags/tags-creating-and-managing",
		"createTime": gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
	}
}

func gcpResourceManagerV3Operation(operationID string, done bool, response map[string]any, responseTypeURL string) map[string]any {
	op := map[string]any{
		"name": "operations/" + strings.TrimSpace(operationID),
		"done": done,
		"metadata": map[string]any{
			"@type":      "type.googleapis.com/google.cloud.resourcemanager.v3.OperationMetadata",
			"createTime": gcpResourceManagerReferenceTime.Format(gcpResourceManagerTimeRFC3339),
		},
	}
	if done && len(response) > 0 {
		if responseTypeURL != "" {
			response["@type"] = responseTypeURL
		}
		op["response"] = response
	}
	return op
}

func gcpResourceManagerV3IAMPolicy(resource string, incoming map[string]any) map[string]any {
	bindRole := "roles/viewer"
	switch {
	case strings.HasPrefix(resource, "folders/"):
		bindRole = "roles/resourcemanager.folderViewer"
	case strings.HasPrefix(resource, "projects/"):
		bindRole = "roles/resourcemanager.projectViewer"
	case strings.HasPrefix(resource, "organizations/"):
		bindRole = "roles/resourcemanager.organizationViewer"
	case strings.HasPrefix(resource, "tagKeys/"):
		bindRole = "roles/resourcemanager.tagAdmin"
	case strings.HasPrefix(resource, "tagValues/"):
		bindRole = "roles/resourcemanager.tagUser"
	}
	bindings := []any{
		map[string]any{
			"role":    bindRole,
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
		"etag":     "cmVzb3VyY2VtYW5hZ2VyLXYzLWV0YWc=",
		"bindings": bindings,
	}
}

const gcpResourceManagerTimeRFC3339 = "2006-01-02T15:04:05Z07:00"
