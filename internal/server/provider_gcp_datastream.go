package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDatastreamRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_datastream(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}
	if !isGCPDatastreamPath(path) {
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

func isGCPDatastreamPath(path string) bool {
	return strings.Contains(path, "/connectionProfiles") ||
		strings.Contains(path, ":discoverConnectionProfile") ||
		strings.Contains(path, "/streams") ||
		strings.Contains(path, "/objects") ||
		strings.Contains(path, ":lookup") ||
		strings.Contains(path, ":run") ||
		strings.Contains(path, ":startBackfillJob") ||
		strings.Contains(path, ":stopBackfillJob") ||
		strings.Contains(path, ":fetchStaticIps") ||
		strings.Contains(path, "/privateConnections") ||
		strings.Contains(path, "/routes")
}

func handleGCPContractProbe_datastream(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "datastream") {
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
			"name":     "projects/stackyard/locations/us-central1/datastream/sample",
			"service":  "datastream",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
