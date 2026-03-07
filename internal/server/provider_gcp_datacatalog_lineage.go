package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDataCatalogLineageRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_datacatalog_lineage(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}
	if !isGCPDataCatalogLineagePath(path) {
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

func isGCPDataCatalogLineagePath(path string) bool {
	return strings.Contains(path, ":processOpenLineageRunEvent") ||
		strings.Contains(path, "/processes") ||
		strings.Contains(path, "/runs") ||
		strings.Contains(path, "/lineageEvents") ||
		strings.Contains(path, ":searchLinks") ||
		strings.Contains(path, ":batchSearchLinkProcesses")
}

func handleGCPContractProbe_datacatalog_lineage(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "datacatalog_lineage") {
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
			"name":     "projects/stackyard/locations/us-central1/datacatalog_lineage/sample",
			"service":  "datacatalog_lineage",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
