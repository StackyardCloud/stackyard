package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPIDSRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_ids(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPIDSPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPIDSPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.ids.v1.IDS/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/endpoints") ||
		strings.Contains(path, "/operations/") ||
		strings.HasSuffix(path, ":cancel")
}

func handleGCPContractProbe_ids(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "ids") {
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
			"name":     "projects/stackyard/locations/us-central1/ids/sample",
			"service":  "ids",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
