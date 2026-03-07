package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDataplexRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_dataplex(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}
	if !isGCPDataplexPath(path) {
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

func isGCPDataplexPath(path string) bool {
	return strings.Contains(path, "/lakes") ||
		strings.Contains(path, "/zones") ||
		strings.Contains(path, "/assets") ||
		strings.Contains(path, "/tasks") ||
		strings.Contains(path, "/jobs") ||
		strings.Contains(path, "/environments") ||
		strings.Contains(path, "/sessions") ||
		strings.Contains(path, "/actions") ||
		strings.Contains(path, ":run") ||
		strings.Contains(path, ":cancel")
}

func handleGCPContractProbe_dataplex(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "dataplex") {
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
			"name":     "projects/stackyard/locations/us-central1/dataplex/sample",
			"service":  "dataplex",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
