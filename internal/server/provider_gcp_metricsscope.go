package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPMetricsScopeRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_metricsscope(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPMetricsScopePath(path) {
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

func isGCPMetricsScopePath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.monitoring.metricsscope.v1.MetricsScopes/") {
		return true
	}
	if strings.Contains(path, ":listMetricsScopesByMonitoredProject") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/v1/locations/global/metricsScopes/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/v1/projects/") && strings.Contains(path, "/locations/global/metricsScopes") {
		return true
	}

	return false
}

func handleGCPContractProbe_metricsscope(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "metricsscope") {
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
			"name":     "projects/stackyard/locations/us-central1/metricsscope/sample",
			"service":  "metricsscope",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
