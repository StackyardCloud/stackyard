package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPPrivateCatalogRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_privatecatalog(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPPrivateCatalogPath(path) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func isGCPPrivateCatalogPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.privatecatalog.v1beta1.PrivateCatalog/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1beta1/") {
		return false
	}

	return strings.Contains(path, "/catalogs:search") ||
		strings.Contains(path, "/products:search") ||
		strings.Contains(path, "/versions:search")
}

func handleGCPContractProbe_privatecatalog(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "privatecatalog") {
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
			"name":     "projects/stackyard/locations/us-central1/privatecatalog/sample",
			"service":  "privatecatalog",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
