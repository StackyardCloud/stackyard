package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPOrgPolicyRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_orgpolicy(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPOrgPolicyPath(path) {
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

func isGCPOrgPolicyPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v2/") {
		return false
	}

	return strings.Contains(path, "/constraints") ||
		strings.Contains(path, "/policies") ||
		strings.Contains(path, "/customConstraints") ||
		strings.Contains(path, ":getEffectivePolicy")
}

func handleGCPContractProbe_orgpolicy(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "orgpolicy") {
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
			"name":     "projects/stackyard/locations/us-central1/orgpolicy/sample",
			"service":  "orgpolicy",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
