package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPOsLoginRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_oslogin(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPOsLoginPath(path) {
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

func isGCPOsLoginPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.oslogin.v1.OsLoginService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/users/") {
		return false
	}

	return strings.Contains(path, "/sshPublicKeys") ||
		strings.Contains(path, "/projects/") ||
		strings.Contains(path, "/loginProfile") ||
		strings.Contains(path, ":importSshPublicKey")
}

func handleGCPContractProbe_oslogin(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "oslogin") {
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
			"name":     "projects/stackyard/locations/us-central1/oslogin/sample",
			"service":  "oslogin",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
