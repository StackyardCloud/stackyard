package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDataLabelingRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_datalabeling(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v1beta1/projects/") {
		return false
	}
	if !isGCPDataLabelingPath(path) {
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

func isGCPDataLabelingPath(path string) bool {
	return strings.Contains(path, "/datasets") ||
		strings.Contains(path, ":importData") ||
		strings.Contains(path, ":exportData") ||
		strings.Contains(path, "/dataItems") ||
		strings.Contains(path, "/annotatedDatasets") ||
		strings.Contains(path, "/image:label") ||
		strings.Contains(path, "/video:label") ||
		strings.Contains(path, "/text:label") ||
		strings.Contains(path, "/examples") ||
		strings.Contains(path, "/annotationSpecSets") ||
		strings.Contains(path, "/instructions") ||
		strings.Contains(path, "/evaluations") ||
		strings.Contains(path, "/exampleComparisons:search") ||
		strings.Contains(path, "/evaluationJobs") ||
		strings.Contains(path, ":pause") ||
		strings.Contains(path, ":resume")
}

func handleGCPContractProbe_datalabeling(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "datalabeling") {
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
			"name":     "projects/stackyard/locations/us-central1/datalabeling/sample",
			"service":  "datalabeling",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
