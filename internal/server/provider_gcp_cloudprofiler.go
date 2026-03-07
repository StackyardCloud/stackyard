package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPCloudProfilerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v2/projects/") {
		return false
	}

	if !isGCPCloudProfilerPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPCloudProfilerListProfiles(w, r, path) {
			return true
		}
		if handleGCPCloudProfilerGetProfile(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPCloudProfilerCreateProfile(w, r, path) {
			return true
		}
		if handleGCPCloudProfilerCreateOfflineProfile(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPCloudProfilerUpdateProfile(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPCloudProfilerPath(path string) bool {
	project, tail, ok := parseGCPCloudProfilerProjectTail(path)
	if !ok || project == "" || len(tail) == 0 {
		return false
	}
	return isGCPCloudProfilerProfilesCollectionTail(tail) ||
		isGCPCloudProfilerProfileTail(tail) ||
		isGCPCloudProfilerCreateOfflineTail(tail)
}

func handleGCPCloudProfilerListProfiles(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPCloudProfilerProjectTail(path)
	if !ok || !isGCPCloudProfilerProfilesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPCloudProfilerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudProfilerProfile(project, "stackyard-profile")}
	return respondGCPCloudProfilerList(w, "profiles", items, pageSize, start, path)
}

func handleGCPCloudProfilerGetProfile(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPCloudProfilerProjectTail(path)
	if !ok || !isGCPCloudProfilerProfileTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudProfilerProfile(project, tail[1]))
	return true
}

func handleGCPCloudProfilerCreateProfile(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPCloudProfilerProjectTail(path)
	if !ok || !isGCPCloudProfilerProfilesCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPCloudProfilerJSONBody(w, r, path)
	if !valid {
		return true
	}
	deployment, _ := body["deployment"].(map[string]any)
	if len(deployment) == 0 {
		respondGCPCloudProfilerInvalidArgument(w, path, "deployment is required")
		return true
	}
	profileTypes, _ := body["profileType"].([]any)
	if len(profileTypes) == 0 {
		respondGCPCloudProfilerInvalidArgument(w, path, "profileType must include at least one value")
		return true
	}
	profile := gcpCloudProfilerProfile(project, "stackyard-profile")
	profile["deployment"] = deployment
	profile["profileType"] = fmt.Sprintf("%v", profileTypes[0])
	respondJSON(w, http.StatusOK, profile)
	return true
}

func handleGCPCloudProfilerCreateOfflineProfile(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPCloudProfilerProjectTail(path)
	if !ok || !isGCPCloudProfilerCreateOfflineTail(tail) {
		return false
	}
	body, valid := decodeGCPCloudProfilerJSONBody(w, r, path)
	if !valid {
		return true
	}
	profile := gcpCloudProfilerBodyMap(body, "profile")
	if len(profile) == 0 {
		respondGCPCloudProfilerInvalidArgument(w, path, "profile is required")
		return true
	}
	name := strings.TrimSpace(gcpCloudProfilerString(profile, "name"))
	if name == "" {
		name = fmt.Sprintf("projects/%s/profiles/offline-profile", project)
	}
	profileID := parseGCPCloudProfilerProfileID(name)
	if profileID == "" {
		respondGCPCloudProfilerInvalidArgument(w, path, "profile.name must reference projects/{project}/profiles/{profile}")
		return true
	}
	response := gcpCloudProfilerProfile(project, profileID)
	if profileType := strings.TrimSpace(gcpCloudProfilerString(profile, "profileType")); profileType != "" {
		response["profileType"] = profileType
	}
	if labels, ok := profile["labels"].(map[string]any); ok && len(labels) > 0 {
		response["labels"] = labels
	}
	if deployment, ok := profile["deployment"].(map[string]any); ok && len(deployment) > 0 {
		response["deployment"] = deployment
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPCloudProfilerUpdateProfile(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPCloudProfilerProjectTail(path)
	if !ok || !isGCPCloudProfilerProfileTail(tail) {
		return false
	}
	body, valid := decodeGCPCloudProfilerJSONBody(w, r, path)
	if !valid {
		return true
	}
	profile := gcpCloudProfilerBodyMap(body, "profile")
	if len(profile) == 0 {
		respondGCPCloudProfilerInvalidArgument(w, path, "profile is required")
		return true
	}
	if name := strings.TrimSpace(gcpCloudProfilerString(profile, "name")); name != "" && parseGCPCloudProfilerProfileID(name) == "" {
		respondGCPCloudProfilerInvalidArgument(w, path, "profile.name must reference projects/{project}/profiles/{profile}")
		return true
	}
	response := gcpCloudProfilerProfile(project, tail[1])
	if labels, ok := profile["labels"].(map[string]any); ok && len(labels) > 0 {
		response["labels"] = labels
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func parseGCPCloudProfilerProjectTail(path string) (project string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" {
		return "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", nil, false
	}
	tail = parts[4:]
	return project, tail, true
}

func isGCPCloudProfilerProfilesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "profiles"
}

func isGCPCloudProfilerProfileTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "profiles" && strings.TrimSpace(tail[1]) != ""
}

func isGCPCloudProfilerCreateOfflineTail(tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	name, action, found := strings.Cut(strings.TrimSpace(tail[0]), ":")
	return found && name == "profiles" && action == "createOffline"
}

func parseGCPCloudProfilerPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPCloudProfilerInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	start = 0
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPCloudProfilerInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPCloudProfilerList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPCloudProfilerInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPCloudProfilerJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPCloudProfilerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpCloudProfilerBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpCloudProfilerString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func parseGCPCloudProfilerProfileID(name string) string {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || strings.TrimSpace(parts[1]) == "" || parts[2] != "profiles" {
		return ""
	}
	return strings.TrimSpace(parts[3])
}

func gcpCloudProfilerProfile(project, profileID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/profiles/%s", project, profileID),
		"profileType": "CPU",
		"deployment": map[string]any{
			"projectId": project,
			"target":    "stackyard-service",
			"labels": map[string]string{
				"language": "go",
				"region":   "us-central1",
			},
		},
		"createTime": "2026-01-01T00:00:00Z",
		"duration":   "60s",
		"labels": map[string]string{
			"source": "stackyard",
		},
	}
}

func respondGCPCloudProfilerInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
