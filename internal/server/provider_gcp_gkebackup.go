package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPGKEBackupRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_gkebackup(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPGKEBackupPath(path) {
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

func isGCPGKEBackupPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.gkebackup.v1.BackupForGKE/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if isGCPGKEBackupLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/backupPlans") ||
		strings.Contains(path, "/backupChannels") ||
		strings.Contains(path, "/backupPlanBindings") ||
		strings.Contains(path, "/backups") ||
		strings.Contains(path, "/volumeBackups") ||
		strings.Contains(path, "/restorePlans") ||
		strings.Contains(path, "/restoreChannels") ||
		strings.Contains(path, "/restorePlanBindings") ||
		strings.Contains(path, "/restores") ||
		strings.Contains(path, "/volumeRestores") ||
		strings.Contains(path, ":getBackupIndexDownloadUrl") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":testIamPermissions") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":cancel")
}

func isGCPGKEBackupLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_gkebackup(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "gkebackup") {
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
			"name":     "projects/stackyard/locations/us-central1/gkebackup/sample",
			"service":  "gkebackup",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
