package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPEventarcRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_eventarc(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPEventarcPath(path) {
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

func isGCPEventarcPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.eventarc.v1.Eventarc/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/triggers") ||
		strings.Contains(path, "/channels") ||
		strings.Contains(path, "/providers") ||
		strings.Contains(path, "/channelConnections") ||
		strings.Contains(path, "/googleChannelConfig") ||
		strings.Contains(path, "/messageBuses") ||
		strings.Contains(path, ":listEnrollments") ||
		strings.Contains(path, "/enrollments") ||
		strings.Contains(path, "/pipelines") ||
		strings.Contains(path, "/googleApiSources") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":testIamPermissions") ||
		strings.Contains(path, ":cancel")
}

func handleGCPContractProbe_eventarc(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "eventarc") {
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
			"name":     "projects/stackyard/locations/us-central1/eventarc/sample",
			"service":  "eventarc",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
