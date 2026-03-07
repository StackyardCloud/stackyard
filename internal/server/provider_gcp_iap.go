package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPIAPRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_iap(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPIAPPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPIAPPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.iap.v1.IdentityAwareProxyAdminService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.iap.v1.IdentityAwareProxyOAuthService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/") {
		return false
	}

	isIAMAction := strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":testIamPermissions")
	if isIAMAction && (strings.Contains(path, "/iap_") ||
		strings.Contains(path, "/brands/") ||
		strings.Contains(path, "/identityAwareProxyClients/")) {
		return true
	}

	return strings.Contains(path, ":iapSettings") ||
		strings.Contains(path, ":validateAttributeExpression") ||
		strings.Contains(path, "/iap_tunnel/") ||
		strings.Contains(path, "/destGroups") ||
		strings.Contains(path, "/brands/") ||
		strings.HasSuffix(path, "/brands") ||
		strings.Contains(path, "/identityAwareProxyClients/") ||
		strings.HasSuffix(path, "/identityAwareProxyClients") ||
		strings.Contains(path, ":resetSecret")
}

func handleGCPContractProbe_iap(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "iap") {
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
			"name":     "projects/stackyard/locations/us-central1/iap/sample",
			"service":  "iap",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
