package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPIdentityToolkitRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_identitytoolkit(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPIdentityToolkitPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodPost:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPIdentityToolkitPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.identitytoolkit.v2.AccountManagementService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.identitytoolkit.v2.AuthenticationService/") {
		return true
	}

	if !strings.HasPrefix(path, "/gcp/v2/accounts/") {
		return false
	}

	return strings.HasSuffix(path, "mfaEnrollment:start") ||
		strings.HasSuffix(path, "mfaEnrollment:finalize") ||
		strings.HasSuffix(path, "mfaEnrollment:withdraw") ||
		strings.HasSuffix(path, "mfaSignIn:start") ||
		strings.HasSuffix(path, "mfaSignIn:finalize")
}

func handleGCPContractProbe_identitytoolkit(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "identitytoolkit") {
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
			"name":     "projects/stackyard/locations/us-central1/identitytoolkit/sample",
			"service":  "identitytoolkit",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
