package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPCloudFunctionsV2Router(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_functions_v2(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPCloudFunctionsV2Path(path) {
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

func isGCPCloudFunctionsV2Path(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.functions.v2.FunctionService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v2/projects/") {
		return false
	}
	if isGCPCloudFunctionsV2LocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/functions") ||
		strings.Contains(path, "/runtimes") ||
		strings.Contains(path, ":generateUploadUrl") ||
		strings.Contains(path, ":generateDownloadUrl") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":testIamPermissions") ||
		strings.Contains(path, "/operations")
}

func isGCPCloudFunctionsV2LocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_functions_v2(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "functions_v2") {
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
			"name":     "projects/stackyard/locations/us-central1/functions_v2/sample",
			"service":  "functions_v2",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
