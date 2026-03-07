package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPPrivilegedAccessManagerRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_privilegedaccessmanager(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPPrivilegedAccessManagerPath(path) {
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

func isGCPPrivilegedAccessManagerPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/") || !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, ":checkOnboardingStatus") ||
		strings.Contains(path, "/entitlements") ||
		strings.Contains(path, "/grants") ||
		strings.Contains(path, ":approve") ||
		strings.Contains(path, ":deny") ||
		strings.Contains(path, ":revoke")
}

func handleGCPContractProbe_privilegedaccessmanager(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "privilegedaccessmanager") {
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
			"name":     "projects/stackyard/locations/us-central1/privilegedaccessmanager/sample",
			"service":  "privilegedaccessmanager",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
