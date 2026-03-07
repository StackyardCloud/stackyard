package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDataCatalogRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_datacatalog(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v1/") {
		return false
	}
	if !isGCPDataCatalogPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodPut:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPDataCatalogPath(path string) bool {
	if strings.Contains(path, "/catalog:search") ||
		strings.Contains(path, "/entryGroups") ||
		strings.Contains(path, "/entries:lookup") ||
		strings.Contains(path, "/tagTemplates") ||
		strings.Contains(path, ":modifyEntryOverview") ||
		strings.Contains(path, ":modifyEntryContacts") ||
		strings.Contains(path, "/tags:reconcile") ||
		strings.Contains(path, ":star") ||
		strings.Contains(path, ":unstar") ||
		strings.Contains(path, ":setConfig") ||
		strings.Contains(path, ":retrieveConfig") ||
		strings.Contains(path, ":retrieveEffectiveConfig") ||
		strings.Contains(path, "/taxonomies") ||
		strings.Contains(path, "/policyTags") ||
		strings.Contains(path, "/taxonomies:import") ||
		strings.Contains(path, "/taxonomies:export") {
		return true
	}

	if strings.Contains(path, "/entries/") && strings.Contains(path, "/tags") {
		return true
	}

	if (strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":testIamPermissions")) &&
		(strings.Contains(path, "/entryGroups/") ||
			strings.Contains(path, "/entries/") ||
			strings.Contains(path, "/tagTemplates/") ||
			strings.Contains(path, "/taxonomies/") ||
			strings.Contains(path, "/policyTags/")) {
		return true
	}

	return false
}

func handleGCPContractProbe_datacatalog(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "datacatalog") {
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
			"name":     "projects/stackyard/locations/us-central1/datacatalog/sample",
			"service":  "datacatalog",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
