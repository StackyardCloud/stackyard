package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPLifeSciencesRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_lifesciences(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPLifeSciencesPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPLifeSciencesPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.lifesciences.v2beta.WorkflowsServiceV2Beta/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v2beta/projects/") {
		return false
	}
	if isGCPLifeSciencesLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/pipelines:run") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":cancel")
}

func isGCPLifeSciencesLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v2beta" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_lifesciences(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "lifesciences") {
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
			"name":     "projects/stackyard/locations/us-central1/lifesciences/sample",
			"service":  "lifesciences",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
