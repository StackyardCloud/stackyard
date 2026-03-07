package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var gcpSecretManagerReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSecretManagerRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_secretmanager(w, r) {
		return true
	}

	path := normalizeGCPSecretManagerPath(rawRequestPath(r))
	if isGCPSecretManagerLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPSecretManagerListLocations(w, r, path) {
			return true
		}
		if handleGCPSecretManagerGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPSecretManagerPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPSecretManagerListSecrets(w, r, path) {
			return true
		}
		if handleGCPSecretManagerGetSecret(w, path) {
			return true
		}
		if handleGCPSecretManagerListSecretVersions(w, r, path) {
			return true
		}
		if handleGCPSecretManagerGetSecretVersion(w, path) {
			return true
		}
		if handleGCPSecretManagerAccessSecretVersion(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPSecretManagerCreateSecret(w, r, path) {
			return true
		}
		if handleGCPSecretManagerAddSecretVersion(w, r, path) {
			return true
		}
		if handleGCPSecretManagerDisableSecretVersion(w, path) {
			return true
		}
		if handleGCPSecretManagerEnableSecretVersion(w, path) {
			return true
		}
		if handleGCPSecretManagerDestroySecretVersion(w, path) {
			return true
		}
		if handleGCPSecretManagerSetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPSecretManagerGetIAMPolicy(w, path) {
			return true
		}
		if handleGCPSecretManagerTestIAMPermissions(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPSecretManagerUpdateSecret(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPSecretManagerDeleteSecret(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSecretManagerPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSecretManagerHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "secretmanager", "secretmanager-apiv1", "secret-manager", "secret_manager", "gcp-secret-manager":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-secretmanager-apiv1") || strings.Contains(ua, "cloud.google.com/go/secretmanager")
}

func isGCPSecretManagerLocationRequest(r *http.Request, path string) bool {
	if !hasGCPSecretManagerHint(r) {
		return false
	}
	_, _, _, ok := parseGCPSecretManagerProjectLocationPath(path)
	return ok
}

func isGCPSecretManagerPath(path string) bool {
	if _, ok := parseGCPSecretManagerSecretsCollectionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPSecretManagerSecretPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPSecretManagerSecretVersionsCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPSecretManagerSecretVersionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPSecretManagerSecretVersionActionPath(path); ok {
		return true
	}
	_, _, _, ok := parseGCPSecretManagerSecretIAMActionPath(path)
	return ok
}

func handleGCPSecretManagerListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPSecretManagerProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPSecretManagerPagination(w, r, path, 500)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecretManagerLocation(project, "global"),
		gcpSecretManagerLocation(project, "us-central1"),
	}
	return respondGCPSecretManagerList(w, "locations", items, pageSize, start, path)
}

func handleGCPSecretManagerGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPSecretManagerProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecretManagerLocation(project, location))
	return true
}

func handleGCPSecretManagerListSecrets(w http.ResponseWriter, r *http.Request, path string) bool {
	project, ok := parseGCPSecretManagerSecretsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecretManagerPagination(w, r, path, 500)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecretManagerSecret(project, "secret-1"),
		gcpSecretManagerSecret(project, "secret-2"),
	}
	return respondGCPSecretManagerList(w, "secrets", items, pageSize, start, path)
}

func handleGCPSecretManagerGetSecret(w http.ResponseWriter, path string) bool {
	project, secretID, ok := parseGCPSecretManagerSecretPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecretManagerSecret(project, secretID))
	return true
}

func handleGCPSecretManagerCreateSecret(w http.ResponseWriter, r *http.Request, path string) bool {
	project, ok := parseGCPSecretManagerSecretsCollectionPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPSecretManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	secret := gcpSecretManagerSecretFromBody(body)
	if len(secret) == 0 {
		respondGCPSecretManagerInvalidArgument(w, path, "secret is required")
		return true
	}
	if !validateGCPSecretManagerSecret(w, path, secret, false) {
		return true
	}

	nameFromBody := gcpSecretManagerString(secret, "name")
	secretIDFromBody := ""
	if nameFromBody != "" {
		bodyProject, bodySecretID, nameOK := parseGCPSecretManagerSecretResourceName(nameFromBody)
		if !nameOK {
			respondGCPSecretManagerInvalidArgument(w, path, "secret.name is invalid")
			return true
		}
		if bodyProject != project {
			respondGCPSecretManagerInvalidArgument(w, path, "secret.name must match parent")
			return true
		}
		secretIDFromBody = bodySecretID
	}

	secretID := strings.TrimSpace(r.URL.Query().Get("secretId"))
	if secretID != "" {
		if !isGCPSecretManagerID(secretID) {
			respondGCPSecretManagerInvalidArgument(w, path, "secretId is invalid")
			return true
		}
		if secretIDFromBody != "" && secretIDFromBody != secretID {
			respondGCPSecretManagerInvalidArgument(w, path, "secretId must match secret.name")
			return true
		}
	} else if secretIDFromBody != "" {
		secretID = secretIDFromBody
	} else {
		secretID = "secret-1"
	}

	resp := gcpSecretManagerSecret(project, secretID)
	applyGCPSecretManagerSecretOverrides(resp, secret)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecretManagerUpdateSecret(w http.ResponseWriter, r *http.Request, path string) bool {
	project, secretID, ok := parseGCPSecretManagerSecretPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecretManagerInvalidArgument(w, path, "updateMask is required")
		return true
	}

	body, valid := decodeGCPSecretManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	secret := gcpSecretManagerSecretFromBody(body)
	if len(secret) == 0 {
		respondGCPSecretManagerInvalidArgument(w, path, "secret is required")
		return true
	}
	if !validateGCPSecretManagerSecret(w, path, secret, true) {
		return true
	}

	expectedName := gcpSecretManagerSecretName(project, secretID)
	if name := gcpSecretManagerString(secret, "name"); name == "" || name != expectedName {
		respondGCPSecretManagerInvalidArgument(w, path, "secret.name must match requested resource")
		return true
	}

	resp := gcpSecretManagerSecret(project, secretID)
	applyGCPSecretManagerSecretOverrides(resp, secret)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecretManagerDeleteSecret(w http.ResponseWriter, path string) bool {
	_, _, ok := parseGCPSecretManagerSecretPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSecretManagerListSecretVersions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, secretID, _, ok := parseGCPSecretManagerSecretVersionsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecretManagerPagination(w, r, path, 500)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecretManagerSecretVersion(project, secretID, "1"),
		gcpSecretManagerSecretVersion(project, secretID, "2"),
		gcpSecretManagerSecretVersion(project, secretID, "3"),
	}
	return respondGCPSecretManagerList(w, "versions", items, pageSize, start, path)
}

func handleGCPSecretManagerGetSecretVersion(w http.ResponseWriter, path string) bool {
	project, secretID, versionID, _, ok := parseGCPSecretManagerSecretVersionPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecretManagerSecretVersion(project, secretID, normalizeGCPSecretManagerVersionID(versionID)))
	return true
}

func handleGCPSecretManagerAddSecretVersion(w http.ResponseWriter, r *http.Request, path string) bool {
	project, secretID, _, ok := parseGCPSecretManagerSecretVersionsCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPSecretManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	payload, _ := body["payload"].(map[string]any)
	data := strings.TrimSpace(gcpSecretManagerString(payload, "data"))
	if data == "" {
		respondGCPSecretManagerInvalidArgument(w, path, "payload.data is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecretManagerSecretVersion(project, secretID, "4"))
	return true
}

func handleGCPSecretManagerAccessSecretVersion(w http.ResponseWriter, path string) bool {
	project, secretID, versionID, action, ok := parseGCPSecretManagerSecretVersionActionPath(path)
	if !ok || action != "access" {
		return false
	}
	versionID = normalizeGCPSecretManagerVersionID(versionID)
	state := gcpSecretManagerVersionState(versionID)
	if state == "DESTROYED" {
		respondGCPSecretManagerFailedPrecondition(w, path, "version is destroyed")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": gcpSecretManagerSecretVersionName(project, secretID, versionID),
		"payload": map[string]any{
			"data":       "c3RhY2t5YXJkLXNlY3JldA==",
			"dataCrc32c": "4077700407",
		},
	})
	return true
}

func handleGCPSecretManagerDisableSecretVersion(w http.ResponseWriter, path string) bool {
	project, secretID, versionID, action, ok := parseGCPSecretManagerSecretVersionActionPath(path)
	if !ok || action != "disable" {
		return false
	}
	versionID = normalizeGCPSecretManagerVersionID(versionID)
	if gcpSecretManagerVersionState(versionID) == "DESTROYED" {
		respondGCPSecretManagerFailedPrecondition(w, path, "cannot disable destroyed version")
		return true
	}
	version := gcpSecretManagerSecretVersion(project, secretID, versionID)
	version["state"] = "DISABLED"
	respondJSON(w, http.StatusOK, version)
	return true
}

func handleGCPSecretManagerEnableSecretVersion(w http.ResponseWriter, path string) bool {
	project, secretID, versionID, action, ok := parseGCPSecretManagerSecretVersionActionPath(path)
	if !ok || action != "enable" {
		return false
	}
	versionID = normalizeGCPSecretManagerVersionID(versionID)
	if gcpSecretManagerVersionState(versionID) == "DESTROYED" {
		respondGCPSecretManagerFailedPrecondition(w, path, "cannot enable destroyed version")
		return true
	}
	version := gcpSecretManagerSecretVersion(project, secretID, versionID)
	version["state"] = "ENABLED"
	respondJSON(w, http.StatusOK, version)
	return true
}

func handleGCPSecretManagerDestroySecretVersion(w http.ResponseWriter, path string) bool {
	project, secretID, versionID, action, ok := parseGCPSecretManagerSecretVersionActionPath(path)
	if !ok || action != "destroy" {
		return false
	}
	version := gcpSecretManagerSecretVersion(project, secretID, normalizeGCPSecretManagerVersionID(versionID))
	version["state"] = "DESTROYED"
	version["destroyTime"] = gcpSecretManagerReferenceTime.Add(2 * time.Hour).Format(time.RFC3339)
	respondJSON(w, http.StatusOK, version)
	return true
}

func handleGCPSecretManagerSetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	project, secretID, action, ok := parseGCPSecretManagerSecretIAMActionPath(path)
	if !ok || action != "setIamPolicy" {
		return false
	}
	body, valid := decodeGCPSecretManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy, _ := body["policy"].(map[string]any)
	if len(policy) == 0 {
		respondGCPSecretManagerInvalidArgument(w, path, "policy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecretManagerIAMPolicy(project, secretID, policy))
	return true
}

func handleGCPSecretManagerGetIAMPolicy(w http.ResponseWriter, path string) bool {
	project, secretID, action, ok := parseGCPSecretManagerSecretIAMActionPath(path)
	if !ok || action != "getIamPolicy" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecretManagerIAMPolicy(project, secretID, nil))
	return true
}

func handleGCPSecretManagerTestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, secretID, action, ok := parseGCPSecretManagerSecretIAMActionPath(path)
	if !ok || action != "testIamPermissions" {
		return false
	}
	body, valid := decodeGCPSecretManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	permissions, _ := body["permissions"].([]any)
	items := make([]string, 0, len(permissions))
	for _, raw := range permissions {
		p, _ := raw.(string)
		if strings.TrimSpace(p) != "" {
			items = append(items, strings.TrimSpace(p))
		}
	}
	if len(items) == 0 {
		items = []string{"secretmanager.secrets.get"}
	}
	_ = project
	_ = secretID
	respondJSON(w, http.StatusOK, map[string]any{
		"permissions": items,
	})
	return true
}

func decodeGCPSecretManagerJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPSecretManagerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		respondGCPSecretManagerInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPSecretManagerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func gcpSecretManagerSecretFromBody(body map[string]any) map[string]any {
	if nested, ok := body["secret"].(map[string]any); ok && len(nested) > 0 {
		return nested
	}
	return body
}

func validateGCPSecretManagerSecret(w http.ResponseWriter, path string, secret map[string]any, requireName bool) bool {
	if requireName && strings.TrimSpace(gcpSecretManagerString(secret, "name")) == "" {
		respondGCPSecretManagerInvalidArgument(w, path, "secret.name is required")
		return false
	}
	replication, _ := secret["replication"].(map[string]any)
	if len(replication) == 0 {
		respondGCPSecretManagerInvalidArgument(w, path, "secret.replication is required")
		return false
	}
	return true
}

func applyGCPSecretManagerSecretOverrides(out, in map[string]any) {
	for _, key := range []string{"labels", "rotation", "etag"} {
		if val, ok := in[key]; ok {
			out[key] = val
		}
	}
	if replication, ok := in["replication"]; ok {
		out["replication"] = replication
	}
}

func parseGCPSecretManagerPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > maxPageSize {
			respondGCPSecretManagerInvalidArgument(w, path, fmt.Sprintf("pageSize must be a non-negative integer <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = n
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			respondGCPSecretManagerInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = n
	}
	return pageSize, start, true
}

func respondGCPSecretManagerList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSecretManagerInvalidArgument(w, path, "pageToken out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextToken := ""
	if end < len(items) {
		nextToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		field:           items[start:end],
		"nextPageToken": nextToken,
	})
	return true
}

func parseGCPSecretManagerProjectLocationPath(path string) (project, location string, list bool, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 5 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && parts[4] == "locations" {
		project = strings.TrimSpace(parts[3])
		if project == "" {
			return "", "", false, false
		}
		return project, "", true, true
	}
	if len(parts) == 6 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && parts[4] == "locations" {
		project = strings.TrimSpace(parts[3])
		location = strings.TrimSpace(parts[5])
		if project == "" || location == "" {
			return "", "", false, false
		}
		return project, location, false, true
	}
	return "", "", false, false
}

func parseGCPSecretManagerProjectTail(path string) (project string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", nil, false
	}
	return project, parts[4:], true
}

func parseGCPSecretManagerSecretsCollectionPath(path string) (project string, ok bool) {
	project, tail, ok := parseGCPSecretManagerProjectTail(path)
	if !ok || len(tail) != 1 || tail[0] != "secrets" {
		return "", false
	}
	return project, true
}

func parseGCPSecretManagerSecretPath(path string) (project, secretID string, ok bool) {
	project, tail, ok := parseGCPSecretManagerProjectTail(path)
	if !ok || len(tail) != 2 || tail[0] != "secrets" {
		return "", "", false
	}
	resource, _, hasAction := gcpSecretManagerResourceActionSegment(tail[1])
	if hasAction || !isGCPSecretManagerID(resource) {
		return "", "", false
	}
	return project, resource, true
}

func parseGCPSecretManagerSecretVersionsCollectionPath(path string) (project, secretID, collection string, ok bool) {
	project, tail, ok := parseGCPSecretManagerProjectTail(path)
	if !ok || len(tail) != 3 || tail[0] != "secrets" || tail[2] != "versions" {
		return "", "", "", false
	}
	if !isGCPSecretManagerID(tail[1]) {
		return "", "", "", false
	}
	return project, tail[1], tail[2], true
}

func parseGCPSecretManagerSecretVersionPath(path string) (project, secretID, versionID, action string, ok bool) {
	project, tail, ok := parseGCPSecretManagerProjectTail(path)
	if !ok || len(tail) != 4 || tail[0] != "secrets" || tail[2] != "versions" {
		return "", "", "", "", false
	}
	if !isGCPSecretManagerID(tail[1]) {
		return "", "", "", "", false
	}
	resource, action, hasAction := gcpSecretManagerResourceActionSegment(tail[3])
	if hasAction {
		return "", "", "", "", false
	}
	if !isGCPSecretManagerVersionID(resource) {
		return "", "", "", "", false
	}
	return project, tail[1], resource, action, true
}

func parseGCPSecretManagerSecretVersionActionPath(path string) (project, secretID, versionID, action string, hasAction bool) {
	project, tail, ok := parseGCPSecretManagerProjectTail(path)
	if !ok || len(tail) != 4 || tail[0] != "secrets" || tail[2] != "versions" {
		return "", "", "", "", false
	}
	if !isGCPSecretManagerID(tail[1]) {
		return "", "", "", "", false
	}
	resource, action, hasAction := gcpSecretManagerResourceActionSegment(tail[3])
	if !hasAction || !isGCPSecretManagerVersionID(resource) {
		return "", "", "", "", false
	}
	return project, tail[1], resource, action, true
}

func parseGCPSecretManagerSecretIAMActionPath(path string) (project, secretID, action string, ok bool) {
	project, tail, ok := parseGCPSecretManagerProjectTail(path)
	if !ok || len(tail) != 2 || tail[0] != "secrets" {
		return "", "", "", false
	}
	resource, action, hasAction := gcpSecretManagerResourceActionSegment(tail[1])
	if !hasAction || !isGCPSecretManagerID(resource) {
		return "", "", "", false
	}
	switch action {
	case "setIamPolicy", "getIamPolicy", "testIamPermissions":
		return project, resource, action, true
	default:
		return "", "", "", false
	}
}

func gcpSecretManagerResourceActionSegment(segment string) (resource, action string, hasAction bool) {
	parts := strings.SplitN(strings.TrimSpace(segment), ":", 2)
	resource = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		action = strings.TrimSpace(parts[1])
		hasAction = action != ""
	}
	return resource, action, hasAction
}

func parseGCPSecretManagerSecretResourceName(name string) (project, secretID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "secrets" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	secretID = strings.TrimSpace(parts[3])
	if project == "" || !isGCPSecretManagerID(secretID) {
		return "", "", false
	}
	return project, secretID, true
}

func isGCPSecretManagerID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func isGCPSecretManagerVersionID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "latest" {
		return true
	}
	if id == "" {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func normalizeGCPSecretManagerVersionID(versionID string) string {
	if strings.EqualFold(strings.TrimSpace(versionID), "latest") {
		return "1"
	}
	return strings.TrimSpace(versionID)
}

func gcpSecretManagerString(body map[string]any, key string) string {
	raw, ok := body[key]
	if !ok {
		return ""
	}
	str, _ := raw.(string)
	return strings.TrimSpace(str)
}

func gcpSecretManagerSecretName(project, secretID string) string {
	return fmt.Sprintf("projects/%s/secrets/%s", project, secretID)
}

func gcpSecretManagerSecretVersionName(project, secretID, versionID string) string {
	return fmt.Sprintf("projects/%s/secrets/%s/versions/%s", project, secretID, versionID)
}

func gcpSecretManagerVersionState(versionID string) string {
	switch strings.TrimSpace(versionID) {
	case "2":
		return "DISABLED"
	case "3":
		return "DESTROYED"
	default:
		return "ENABLED"
	}
}

func gcpSecretManagerLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(strings.ReplaceAll(location, "-", " ")),
		"labels": map[string]any{
			"cloud.googleapis.com/region": location,
		},
		"metadata": map[string]any{
			"provider": providerGCP,
			"service":  "secretmanager",
		},
	}
}

func gcpSecretManagerSecret(project, secretID string) map[string]any {
	return map[string]any{
		"name":       gcpSecretManagerSecretName(project, secretID),
		"createTime": gcpSecretManagerReferenceTime.Format(time.RFC3339),
		"replication": map[string]any{
			"automatic": map[string]any{},
		},
		"labels": map[string]any{
			"env": "staged",
		},
		"rotation": map[string]any{
			"nextRotationTime": gcpSecretManagerReferenceTime.Add(24 * time.Hour).Format(time.RFC3339),
			"rotationPeriod":   "86400s",
		},
		"etag": "etag-" + secretID,
	}
}

func gcpSecretManagerSecretVersion(project, secretID, versionID string) map[string]any {
	versionID = normalizeGCPSecretManagerVersionID(versionID)
	out := map[string]any{
		"name":       gcpSecretManagerSecretVersionName(project, secretID, versionID),
		"state":      gcpSecretManagerVersionState(versionID),
		"createTime": gcpSecretManagerReferenceTime.Add(30 * time.Second).Format(time.RFC3339),
		"etag":       "etag-version-" + versionID,
	}
	if out["state"] == "DESTROYED" {
		out["destroyTime"] = gcpSecretManagerReferenceTime.Add(2 * time.Hour).Format(time.RFC3339)
	}
	return out
}

func gcpSecretManagerIAMPolicy(project, secretID string, in map[string]any) map[string]any {
	resource := gcpSecretManagerSecretName(project, secretID)
	if in != nil {
		policy := map[string]any{
			"version": 1,
			"bindings": []any{
				map[string]any{
					"role":    "roles/secretmanager.secretAccessor",
					"members": []any{"user:stackyard@example.com"},
				},
			},
			"etag": "policy-etag-" + secretID,
		}
		for _, key := range []string{"version", "bindings", "etag"} {
			if val, ok := in[key]; ok {
				policy[key] = val
			}
		}
		return policy
	}
	return map[string]any{
		"version": 1,
		"bindings": []any{
			map[string]any{
				"role":    "roles/secretmanager.secretAccessor",
				"members": []any{"user:stackyard@example.com"},
			},
		},
		"etag":     "policy-etag-" + secretID,
		"resource": resource,
	}
}

func respondGCPSecretManagerInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSecretManagerError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSecretManagerFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSecretManagerError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSecretManagerError(w http.ResponseWriter, status int, err, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    err,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_secretmanager(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "secretmanager") {
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
			"name":     "projects/stackyard/secrets/sample-secret",
			"service":  "secretmanager",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
