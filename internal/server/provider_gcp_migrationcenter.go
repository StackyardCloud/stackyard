package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPMigrationCenterRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_migrationcenter(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPMigrationCenterPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPMigrationCenterPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.migrationcenter.v1.MigrationCenter/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if isGCPMigrationCenterLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/assets") ||
		strings.Contains(path, "/importJobs") ||
		strings.Contains(path, "/importDataFiles") ||
		strings.Contains(path, "/groups") ||
		strings.Contains(path, "/errorFrames") ||
		strings.Contains(path, "/sources") ||
		strings.Contains(path, "/preferenceSets") ||
		strings.Contains(path, "/settings") ||
		strings.Contains(path, "/reportConfigs") ||
		strings.Contains(path, "/reports") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":batchUpdate") ||
		strings.Contains(path, ":batchDelete") ||
		strings.Contains(path, ":reportAssetFrames") ||
		strings.Contains(path, ":aggregateValues") ||
		strings.Contains(path, ":validate") ||
		strings.Contains(path, ":run") ||
		strings.Contains(path, ":addAssets") ||
		strings.Contains(path, ":removeAssets") ||
		strings.Contains(path, ":cancel")
}

func isGCPMigrationCenterLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_migrationcenter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "migrationcenter") {
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
			"name":     "projects/stackyard/locations/us-central1/migrationcenter/sample",
			"service":  "migrationcenter",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
