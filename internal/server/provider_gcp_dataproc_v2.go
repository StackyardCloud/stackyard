package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDataprocV2Router(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_dataproc_v2(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !isGCPDataprocV2Path(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPDataprocV2Path(path string) bool {
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/sessions") ||
		strings.Contains(path, "/sessionTemplates") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, "/batches") ||
		strings.Contains(path, ":terminate") ||
		strings.Contains(path, ":cancel") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":testIamPermissions")
}

func handleGCPContractProbe_dataproc_v2(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "dataproc_v2") {
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
			"name":     "projects/stackyard/locations/us-central1/dataproc_v2/sample",
			"service":  "dataproc_v2",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
