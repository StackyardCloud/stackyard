package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPErrorReportingRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_errorreporting(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPErrorReportingPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPErrorReportingPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.devtools.clouderrorreporting.v1beta1.ErrorStatsService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.devtools.clouderrorreporting.v1beta1.ErrorGroupService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.devtools.clouderrorreporting.v1beta1.ReportErrorsService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1beta1/projects/") {
		return false
	}

	return strings.Contains(path, "/groupStats") ||
		strings.Contains(path, "/events") ||
		strings.Contains(path, "/groups/") ||
		strings.Contains(path, ":report")
}

func handleGCPContractProbe_errorreporting(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "errorreporting") {
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
			"name":     "projects/stackyard/locations/us-central1/errorreporting/sample",
			"service":  "errorreporting",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
