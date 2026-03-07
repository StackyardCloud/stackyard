package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPMemorystoreRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_memorystore(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPMemorystorePath(path) {
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

func isGCPMemorystorePath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.memorystore.v1.Memorystore/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if isGCPMemorystoreLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/instances") ||
		strings.Contains(path, "/certificateAuthority") ||
		strings.Contains(path, ":rescheduleMaintenance") ||
		strings.Contains(path, "/backupCollections") ||
		strings.Contains(path, "/backups") ||
		strings.Contains(path, ":export") ||
		strings.Contains(path, ":backup") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":cancel")
}

func isGCPMemorystoreLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_memorystore(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "memorystore") {
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
			"name":     "projects/stackyard/locations/us-central1/memorystore/sample",
			"service":  "memorystore",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
