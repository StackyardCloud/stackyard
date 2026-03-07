package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDataQNARouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_dataqna(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v1alpha/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}
	if !isGCPDataQnAPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPatch:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPDataQnAPath(path string) bool {
	return strings.Contains(path, ":suggestQueries") ||
		strings.Contains(path, "/questions") ||
		strings.Contains(path, ":execute") ||
		strings.Contains(path, "/userFeedback")
}

func handleGCPContractProbe_dataqna(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "dataqna") {
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
			"name":     "projects/stackyard/locations/us-central1/dataqna/sample",
			"service":  "dataqna",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
