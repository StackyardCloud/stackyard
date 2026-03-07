package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPMetastoreRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_metastore(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPMetastorePath(path) {
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

func isGCPMetastorePath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.metastore.v1.DataprocMetastore/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if isGCPMetastoreLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/services") ||
		strings.Contains(path, "/metadataImports") ||
		strings.Contains(path, "/backups") ||
		strings.Contains(path, ":exportMetadata") ||
		strings.Contains(path, ":restore") ||
		strings.Contains(path, ":queryMetadata") ||
		strings.Contains(path, ":moveTableToDatabase") ||
		strings.Contains(path, ":alterLocation") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":testIamPermissions") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":cancel")
}

func isGCPMetastoreLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_metastore(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "metastore") {
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
			"name":     "projects/stackyard/locations/us-central1/metastore/sample",
			"service":  "metastore",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
