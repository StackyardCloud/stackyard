package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDataFusionRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_datafusion(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !isGCPDataFusionPath(path) {
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

func isGCPDataFusionPath(path string) bool {
	return strings.Contains(path, "/versions") ||
		strings.Contains(path, "/instances") ||
		strings.Contains(path, ":restart")
}

func handleGCPContractProbe_datafusion(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "datafusion") {
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
			"name":     "projects/stackyard/locations/us-central1/datafusion/sample",
			"service":  "datafusion",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
