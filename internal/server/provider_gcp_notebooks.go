package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPNotebooksRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_notebooks(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPNotebooksPath(path) {
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

func isGCPNotebooksPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.notebooks.v1.NotebookService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.notebooks.v1.ManagedNotebookService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/instances") ||
		strings.Contains(path, "/environments") ||
		strings.Contains(path, "/schedules") ||
		strings.Contains(path, "/executions") ||
		strings.Contains(path, "/runtimes")
}

func handleGCPContractProbe_notebooks(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "notebooks") {
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
			"name":     "projects/stackyard/locations/us-central1/notebooks/sample",
			"service":  "notebooks",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
