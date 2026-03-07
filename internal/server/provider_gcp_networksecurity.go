package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPNetworkSecurityRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_networksecurity(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPNetworkSecurityPath(path) {
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

func isGCPNetworkSecurityPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.networksecurity.v1beta1.NetworkSecurity/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1beta1/projects/") || !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/authorizationPolicies") ||
		strings.Contains(path, "/clientTlsPolicies") ||
		strings.Contains(path, "/serverTlsPolicies")
}

func handleGCPContractProbe_networksecurity(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "networksecurity") {
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
			"name":     "projects/stackyard/locations/us-central1/networksecurity/sample",
			"service":  "networksecurity",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
