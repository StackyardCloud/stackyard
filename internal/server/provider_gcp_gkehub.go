package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPGKEHubRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_gkehub(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPGKEHubPath(path) {
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

func isGCPGKEHubPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.gkehub.v1beta1.GkeHubMembershipService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1beta1/projects/") {
		return false
	}
	if isGCPGKEHubLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/memberships") ||
		strings.Contains(path, ":generateConnectManifest") ||
		strings.Contains(path, "memberships:validateExclusivity") ||
		strings.Contains(path, ":generateExclusivityManifest") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":testIamPermissions") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":cancel")
}

func isGCPGKEHubLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1beta1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_gkehub(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "gkehub") {
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
			"name":     "projects/stackyard/locations/us-central1/gkehub/sample",
			"service":  "gkehub",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
