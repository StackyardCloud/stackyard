package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPIAMAdminRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPIAMAdminPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.iam.admin.v1.IAM/") {
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
		if handleGCPIAMAdminListServiceAccounts(w, r, path) {
			return true
		}
		if handleGCPIAMAdminGetServiceAccount(w, path) {
			return true
		}
		if handleGCPIAMAdminListRoles(w, r, path) {
			return true
		}
		if handleGCPIAMAdminGetRole(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPIAMAdminCreateServiceAccount(w, r, path) {
			return true
		}
		if handleGCPIAMAdminServiceAccountAction(w, path) {
			return true
		}
		if handleGCPIAMAdminCreateRole(w, r, path) {
			return true
		}
		if handleGCPIAMAdminQueryGrantableRoles(w, r, path) {
			return true
		}
		if handleGCPIAMAdminQueryTestablePermissions(w, r, path) {
			return true
		}
		if handleGCPIAMAdminQueryAuditableServices(w, r, path) {
			return true
		}
		if handleGCPIAMAdminLintPolicy(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPIAMAdminPatchServiceAccount(w, r, path) {
			return true
		}
		if handleGCPIAMAdminUpdateRole(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPIAMAdminDeleteServiceAccount(w, path) {
			return true
		}
		if handleGCPIAMAdminDeleteRole(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPIAMAdminPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.iam.admin.v1.IAM/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/") {
		return false
	}

	return strings.Contains(path, "/serviceAccounts") ||
		strings.Contains(path, "/roles") ||
		strings.Contains(path, ":queryGrantableRoles") ||
		strings.Contains(path, ":queryTestablePermissions") ||
		strings.Contains(path, ":queryAuditableServices") ||
		strings.Contains(path, ":lintPolicy")
}

func handleGCPIAMAdminListServiceAccounts(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPIAMAdminProjectTail(path)
	if !ok || len(tail) != 1 || tail[0] != "serviceAccounts" {
		return false
	}
	pageSize, start, valid := parseGCPIAMAdminPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpIAMAdminServiceAccount(project, "stackyard@example.iam.gserviceaccount.com")}
	return respondGCPIAMAdminList(w, "accounts", items, pageSize, start, path)
}

func handleGCPIAMAdminGetServiceAccount(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPIAMAdminProjectTail(path)
	if !ok || len(tail) != 2 || tail[0] != "serviceAccounts" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpIAMAdminServiceAccount(project, tail[1]))
	return true
}

func handleGCPIAMAdminCreateServiceAccount(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPIAMAdminProjectTail(path)
	if !ok || len(tail) != 1 || tail[0] != "serviceAccounts" {
		return false
	}
	body, valid := decodeGCPIAMAdminJSONBody(w, r, path)
	if !valid {
		return true
	}
	accountID := strings.TrimSpace(stringFromMap(body, "accountId"))
	if accountID == "" {
		accountID = strings.TrimSpace(r.URL.Query().Get("accountId"))
	}
	if accountID == "" {
		respondGCPIAMAdminInvalidArgument(w, path, "accountId is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpIAMAdminServiceAccount(project, accountID+"@example.iam.gserviceaccount.com"))
	return true
}

func handleGCPIAMAdminPatchServiceAccount(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPIAMAdminProjectTail(path)
	if !ok || len(tail) != 2 || tail[0] != "serviceAccounts" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	body, valid := decodeGCPIAMAdminJSONBody(w, r, path)
	if !valid {
		return true
	}
	serviceAccount := gcpIAMAdminBodyMap(body, "serviceAccount")
	if len(serviceAccount) == 0 {
		respondGCPIAMAdminInvalidArgument(w, path, "serviceAccount is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpIAMAdminServiceAccount(project, tail[1]))
	return true
}

func handleGCPIAMAdminDeleteServiceAccount(w http.ResponseWriter, path string) bool {
	_, tail, ok := parseGCPIAMAdminProjectTail(path)
	if !ok || len(tail) != 2 || tail[0] != "serviceAccounts" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPIAMAdminServiceAccountAction(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPIAMAdminProjectTail(path)
	if !ok || len(tail) != 2 || tail[0] != "serviceAccounts" {
		return false
	}
	accountID, action, hasAction := strings.Cut(normalizeGCPIAMAdminActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(accountID) == "" {
		return false
	}
	switch action {
	case "disable", "enable":
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	case "undelete":
		respondJSON(w, http.StatusOK, gcpIAMAdminServiceAccount(project, accountID))
		return true
	default:
		return false
	}
}

func handleGCPIAMAdminListRoles(w http.ResponseWriter, r *http.Request, path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "roles" {
		pageSize, start, valid := parseGCPIAMAdminPagination(w, r, path)
		if !valid {
			return true
		}
		items := []map[string]any{gcpIAMAdminRole("projects/stackyard", "customViewer")}
		return respondGCPIAMAdminList(w, "roles", items, pageSize, start, path)
	}

	if len(parts) == 5 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && parts[4] == "roles" {
		project := strings.TrimSpace(parts[3])
		if project == "" {
			return false
		}
		pageSize, start, valid := parseGCPIAMAdminPagination(w, r, path)
		if !valid {
			return true
		}
		items := []map[string]any{gcpIAMAdminRole("projects/"+project, "customViewer")}
		return respondGCPIAMAdminList(w, "roles", items, pageSize, start, path)
	}

	return false
}

func handleGCPIAMAdminGetRole(w http.ResponseWriter, path string) bool {
	parent, roleID, ok := parseGCPIAMAdminRolePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpIAMAdminRole(parent, roleID))
	return true
}

func handleGCPIAMAdminCreateRole(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPIAMAdminProjectTail(path)
	if !ok || len(tail) != 1 || tail[0] != "roles" {
		return false
	}
	body, valid := decodeGCPIAMAdminJSONBody(w, r, path)
	if !valid {
		return true
	}
	role := gcpIAMAdminBodyMap(body, "role")
	if len(role) == 0 {
		respondGCPIAMAdminInvalidArgument(w, path, "role is required")
		return true
	}
	roleID := strings.TrimSpace(stringFromMap(body, "roleId"))
	if roleID == "" {
		roleID = strings.TrimSpace(r.URL.Query().Get("roleId"))
	}
	if roleID == "" {
		respondGCPIAMAdminInvalidArgument(w, path, "roleId is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpIAMAdminRole("projects/"+project, roleID))
	return true
}

func handleGCPIAMAdminUpdateRole(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, roleID, ok := parseGCPIAMAdminRolePath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPIAMAdminJSONBody(w, r, path)
	if !valid {
		return true
	}
	role := gcpIAMAdminBodyMap(body, "role")
	if len(role) == 0 {
		respondGCPIAMAdminInvalidArgument(w, path, "role is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpIAMAdminRole(parent, roleID))
	return true
}

func handleGCPIAMAdminDeleteRole(w http.ResponseWriter, path string) bool {
	_, _, ok := parseGCPIAMAdminRolePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPIAMAdminQueryGrantableRoles(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v1/roles:queryGrantableRoles" {
		return false
	}
	body, valid := decodeGCPIAMAdminJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(stringFromMap(body, "fullResourceName")) == "" {
		respondGCPIAMAdminInvalidArgument(w, path, "fullResourceName is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"roles": []any{gcpIAMAdminRole("projects/stackyard", "customViewer")},
	})
	return true
}

func handleGCPIAMAdminQueryTestablePermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v1/permissions:queryTestablePermissions" {
		return false
	}
	body, valid := decodeGCPIAMAdminJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(stringFromMap(body, "fullResourceName")) == "" {
		respondGCPIAMAdminInvalidArgument(w, path, "fullResourceName is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"permissions": []any{
			map[string]any{"name": "resourcemanager.projects.get", "stage": "GA"},
		},
	})
	return true
}

func handleGCPIAMAdminQueryAuditableServices(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v1/iamPolicies:queryAuditableServices" {
		return false
	}
	body, valid := decodeGCPIAMAdminJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(stringFromMap(body, "fullResourceName")) == "" {
		respondGCPIAMAdminInvalidArgument(w, path, "fullResourceName is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"services": []any{map[string]any{"name": "cloudresourcemanager.googleapis.com"}},
	})
	return true
}

func handleGCPIAMAdminLintPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v1/iamPolicies:lintPolicy" {
		return false
	}
	body, valid := decodeGCPIAMAdminJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(stringFromMap(body, "fullResourceName")) == "" {
		respondGCPIAMAdminInvalidArgument(w, path, "fullResourceName is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"lintResults": []any{},
	})
	return true
}

func parseGCPIAMAdminProjectTail(path string) (project string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", nil, false
	}
	return project, parts[4:], len(parts[4:]) > 0
}

func parseGCPIAMAdminRolePath(path string) (parent, roleID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 || parts[0] != "gcp" || parts[1] != "v1" {
		return "", "", false
	}

	if len(parts) == 6 && parts[2] == "projects" && parts[4] == "roles" {
		project := strings.TrimSpace(parts[3])
		roleID = strings.TrimSpace(parts[5])
		if project == "" || roleID == "" || strings.Contains(roleID, ":") {
			return "", "", false
		}
		return "projects/" + project, roleID, true
	}

	if len(parts) == 4 && parts[2] == "roles" {
		roleID = strings.TrimSpace(parts[3])
		if roleID == "" || strings.Contains(roleID, ":") {
			return "", "", false
		}
		return "roles", roleID, true
	}

	return "", "", false
}

func parseGCPIAMAdminPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPIAMAdminInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPIAMAdminInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPIAMAdminList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPIAMAdminInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPIAMAdminJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPIAMAdminInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpIAMAdminBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func normalizeGCPIAMAdminActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func gcpIAMAdminServiceAccount(project, email string) map[string]any {
	if strings.Contains(email, "@") {
		return map[string]any{
			"name":           fmt.Sprintf("projects/%s/serviceAccounts/%s", project, email),
			"projectId":      project,
			"uniqueId":       "123456789012345678901",
			"email":          email,
			"displayName":    "Stackyard IAM Admin SA",
			"oauth2ClientId": "109876543210987654321",
		}
	}
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/serviceAccounts/%s", project, email),
		"projectId":   project,
		"uniqueId":    "123456789012345678901",
		"email":       email + "@example.iam.gserviceaccount.com",
		"displayName": "Stackyard IAM Admin SA",
	}
}

func gcpIAMAdminRole(parent, roleID string) map[string]any {
	name := parent + "/roles/" + roleID
	if parent == "roles" {
		name = "roles/" + roleID
	}
	return map[string]any{
		"name":                name,
		"title":               "Stackyard Custom Viewer",
		"description":         "Role managed by Stackyard",
		"includedPermissions": []string{"resourcemanager.projects.get"},
		"stage":               "GA",
	}
}

func respondGCPIAMAdminInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
