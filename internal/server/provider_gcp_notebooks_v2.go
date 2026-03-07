package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPNotebooksV2Router(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_notebooks_v2(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPNotebooksV2Path(path) {
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

func isGCPNotebooksV2Path(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.notebooks.v2.NotebookService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v2/projects/") {
		return false
	}
	if isGCPNotebooksV2LocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/instances") ||
		strings.Contains(path, ":start") ||
		strings.Contains(path, ":stop") ||
		strings.Contains(path, ":reset") ||
		strings.Contains(path, ":checkUpgradability") ||
		strings.Contains(path, ":upgrade") ||
		strings.Contains(path, ":rollback") ||
		strings.Contains(path, ":diagnose") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":testIamPermissions") ||
		strings.Contains(path, "/operations")
}

func isGCPNotebooksV2LocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_notebooks_v2(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "notebooks_v2") {
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
			"name":     "projects/stackyard/locations/us-central1/notebooks_v2/sample",
			"service":  "notebooks_v2",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
