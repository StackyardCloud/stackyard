package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPInfrastructureManagerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if isGCPInfrastructureManagerLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPInfrastructureManagerListLocations(w, r, path) {
			return true
		}
		if handleGCPInfrastructureManagerGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPInfrastructureManagerPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPInfrastructureManagerListDeployments(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerGetDeployment(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerExportLockInfo(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerListPreviews(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerGetPreview(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerListTerraformVersions(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerGetTerraformVersion(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerListResourceChanges(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerGetResourceChange(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerListResourceDrifts(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerGetResourceDrift(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPInfrastructureManagerCreateDeployment(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerLockDeployment(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerUnlockDeployment(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerCreatePreview(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerExportPreviewResult(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPInfrastructureManagerDeleteDeployment(w, path) {
			return true
		}
		if handleGCPInfrastructureManagerDeletePreview(w, path) {
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

func isGCPInfrastructureManagerPath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return false
	}
	resource := strings.TrimSpace(parts[6])
	switch resource {
	case "deployments", "previews", "terraformVersions", "resourceChanges", "resourceDrifts":
		return true
	default:
		return false
	}
}

func isGCPInfrastructureManagerLocationRequest(r *http.Request, path string) bool {
	return isGCPProjectLocationDiscoveryPath(path) && hasGCPInfrastructureManagerHint(r)
}

func hasGCPInfrastructureManagerHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	if service == "config" || service == "infrastructuremanager" {
		return true
	}
	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "stackyard-config-apiv1")
}

func handleGCPInfrastructureManagerListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPInfrastructureManagerProjectLocationPath(path)
	if !ok || !list {
		return false
	}

	pageSize, err := parseGCPInfrastructureManagerOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseGCPInfrastructureManagerOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "pageToken must be a non-negative integer offset",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
	}

	locations := []map[string]any{
		gcpInfrastructureManagerLocation(project, "us-central1"),
		gcpInfrastructureManagerLocation(project, "global"),
	}
	if start > len(locations) {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageToken is out of range",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	end := len(locations)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}

	nextPageToken := ""
	if end < len(locations) {
		nextPageToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"locations":     locations[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func handleGCPInfrastructureManagerGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPInfrastructureManagerProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpInfrastructureManagerLocation(project, location))
	return true
}

func gcpInfrastructureManagerLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Infrastructure Manager " + location,
		"labels": map[string]string{
			"service": "config",
			"stage":   "emulated",
		},
	}
}

func handleGCPInfrastructureManagerListDeployments(w http.ResponseWriter, path string) bool {
	if _, _, _, list, ok := parseGCPInfrastructureManagerCollectionPath(path, "deployments"); !ok || !list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"deployments":   []any{},
		"nextPageToken": "",
	})
	return true
}

func handleGCPInfrastructureManagerGetDeployment(w http.ResponseWriter, path string) bool {
	project, location, deployment, list, ok := parseGCPInfrastructureManagerCollectionPath(path, "deployments")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/deployments/%s", project, location, deployment),
	})
	return true
}

func handleGCPInfrastructureManagerCreateDeployment(w http.ResponseWriter, path string) bool {
	project, location, _, list, ok := parseGCPInfrastructureManagerCollectionPath(path, "deployments")
	if !ok || !list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpInfrastructureManagerOperation(project, location, "create-deployment"))
	return true
}

func handleGCPInfrastructureManagerDeleteDeployment(w http.ResponseWriter, path string) bool {
	project, location, _, list, ok := parseGCPInfrastructureManagerCollectionPath(path, "deployments")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpInfrastructureManagerOperation(project, location, "delete-deployment"))
	return true
}

func handleGCPInfrastructureManagerLockDeployment(w http.ResponseWriter, path string) bool {
	project, location, _, ok := parseGCPInfrastructureManagerActionPath(path, "deployments", "lock")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpInfrastructureManagerOperation(project, location, "lock-deployment"))
	return true
}

func handleGCPInfrastructureManagerUnlockDeployment(w http.ResponseWriter, path string) bool {
	project, location, _, ok := parseGCPInfrastructureManagerActionPath(path, "deployments", "unlock")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpInfrastructureManagerOperation(project, location, "unlock-deployment"))
	return true
}

func handleGCPInfrastructureManagerExportLockInfo(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPInfrastructureManagerActionPath(path, "deployments", "exportLock")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPInfrastructureManagerListPreviews(w http.ResponseWriter, path string) bool {
	if _, _, _, list, ok := parseGCPInfrastructureManagerCollectionPath(path, "previews"); !ok || !list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"previews":      []any{},
		"nextPageToken": "",
	})
	return true
}

func handleGCPInfrastructureManagerGetPreview(w http.ResponseWriter, path string) bool {
	project, location, preview, list, ok := parseGCPInfrastructureManagerCollectionPath(path, "previews")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/previews/%s", project, location, preview),
	})
	return true
}

func handleGCPInfrastructureManagerCreatePreview(w http.ResponseWriter, path string) bool {
	project, location, _, list, ok := parseGCPInfrastructureManagerCollectionPath(path, "previews")
	if !ok || !list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpInfrastructureManagerOperation(project, location, "create-preview"))
	return true
}

func handleGCPInfrastructureManagerDeletePreview(w http.ResponseWriter, path string) bool {
	project, location, _, list, ok := parseGCPInfrastructureManagerCollectionPath(path, "previews")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpInfrastructureManagerOperation(project, location, "delete-preview"))
	return true
}

func handleGCPInfrastructureManagerExportPreviewResult(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPInfrastructureManagerActionPath(path, "previews", "export")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPInfrastructureManagerListTerraformVersions(w http.ResponseWriter, path string) bool {
	if _, _, _, list, ok := parseGCPInfrastructureManagerCollectionPath(path, "terraformVersions"); !ok || !list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"terraformVersions": []any{},
		"nextPageToken":     "",
	})
	return true
}

func handleGCPInfrastructureManagerGetTerraformVersion(w http.ResponseWriter, path string) bool {
	project, location, version, list, ok := parseGCPInfrastructureManagerCollectionPath(path, "terraformVersions")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/terraformVersions/%s", project, location, version),
	})
	return true
}

func handleGCPInfrastructureManagerListResourceChanges(w http.ResponseWriter, path string) bool {
	if _, _, _, _, list, ok := parseGCPInfrastructureManagerPreviewChildPath(path, "resourceChanges"); !ok || !list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"resourceChanges": []any{},
		"nextPageToken":   "",
	})
	return true
}

func handleGCPInfrastructureManagerGetResourceChange(w http.ResponseWriter, path string) bool {
	project, location, preview, id, list, ok := parseGCPInfrastructureManagerPreviewChildPath(path, "resourceChanges")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/previews/%s/resourceChanges/%s", project, location, preview, id),
	})
	return true
}

func handleGCPInfrastructureManagerListResourceDrifts(w http.ResponseWriter, path string) bool {
	if _, _, _, _, list, ok := parseGCPInfrastructureManagerPreviewChildPath(path, "resourceDrifts"); !ok || !list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"resourceDrifts": []any{},
		"nextPageToken":  "",
	})
	return true
}

func handleGCPInfrastructureManagerGetResourceDrift(w http.ResponseWriter, path string) bool {
	project, location, preview, id, list, ok := parseGCPInfrastructureManagerPreviewChildPath(path, "resourceDrifts")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/previews/%s/resourceDrifts/%s", project, location, preview, id),
	})
	return true
}

func gcpInfrastructureManagerOperation(project, location, op string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s-op-1", project, location, op),
		"done": true,
	}
}

func parseGCPInfrastructureManagerProjectLocationPath(path string) (project, location string, list, ok bool) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", "", false, false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) != 5 && len(parts) != 6 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", false, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", "", false, false
	}
	if len(parts) == 5 {
		return project, "", true, true
	}
	location = strings.TrimSpace(parts[5])
	if location == "" {
		return "", "", false, false
	}
	return project, location, false, true
}

func parseGCPInfrastructureManagerCollectionPath(path, collection string) (project, location, id string, list, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 7 && len(parts) != 8 {
		return "", "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != collection {
		return "", "", "", false, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", "", false, false
	}
	if len(parts) == 7 {
		return project, location, "", true, true
	}
	id = strings.TrimSpace(parts[7])
	if id == "" || strings.Contains(id, ":") {
		return "", "", "", false, false
	}
	return project, location, id, false, true
}

func parseGCPInfrastructureManagerActionPath(path, collection, action string) (project, location, id string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 8 {
		return "", "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != collection {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", "", false
	}
	idAndAction := strings.TrimSpace(parts[7])
	id, parsedAction, found := strings.Cut(idAndAction, ":")
	if !found || strings.TrimSpace(id) == "" || strings.TrimSpace(parsedAction) != action {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(id), true
}

func parseGCPInfrastructureManagerPreviewChildPath(path, child string) (project, location, preview, id string, list, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 9 && len(parts) != 10 {
		return "", "", "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "previews" || parts[8] != child {
		return "", "", "", "", false, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	preview = strings.TrimSpace(parts[7])
	if project == "" || location == "" || preview == "" || strings.Contains(preview, ":") {
		return "", "", "", "", false, false
	}
	if len(parts) == 9 {
		return project, location, preview, "", true, true
	}
	id = strings.TrimSpace(parts[9])
	if id == "" || strings.Contains(id, ":") {
		return "", "", "", "", false, false
	}
	return project, location, preview, id, false, true
}

func parseGCPInfrastructureManagerOptionalNonNegativeInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("expected non-negative integer")
	}
	return n, nil
}
