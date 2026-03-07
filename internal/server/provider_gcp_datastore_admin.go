package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDatastoreAdminRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_datastore_admin(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPDatastoreAdminPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPDatastoreAdminPath(path string) bool {
	const prefix = "/gcp/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return false
	}

	remainder := strings.TrimPrefix(path, prefix)
	if remainder == "" {
		return false
	}

	projectParts := strings.SplitN(remainder, "/", 2)
	projectSegment := projectParts[0]
	projectAction := ""
	if strings.Contains(projectSegment, ":") {
		parts := strings.SplitN(projectSegment, ":", 2)
		projectSegment = parts[0]
		projectAction = parts[1]
	}

	projectID := projectSegment
	if projectID == "" {
		return false
	}

	if projectAction != "" {
		return projectAction == "export" || projectAction == "import"
	}

	rest := ""
	if len(projectParts) == 2 {
		rest = "/" + projectParts[1]
	}

	return strings.HasPrefix(rest, "/indexes") ||
		strings.HasPrefix(rest, "/operations")
}

func handleGCPContractProbe_datastore_admin(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "datastore_admin") {
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
			"name":     "projects/stackyard/locations/us-central1/datastore_admin/sample",
			"service":  "datastore_admin",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
