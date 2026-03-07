package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPFilestoreRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_filestore(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPFilestorePath(path) {
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

func isGCPFilestorePath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.filestore.v1.CloudFilestoreManager/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if isGCPFilestoreLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/instances") ||
		strings.Contains(path, "/snapshots") ||
		strings.Contains(path, "/backups") ||
		strings.Contains(path, ":restore") ||
		strings.Contains(path, ":revert") ||
		strings.Contains(path, ":promoteReplica") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":cancel")
}

func isGCPFilestoreLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_filestore(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "filestore") {
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
			"name":     "projects/stackyard/locations/us-central1/filestore/sample",
			"service":  "filestore",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
