package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPOsConfigRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_osconfig(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPOsConfigPath(path) {
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

func isGCPOsConfigPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.osconfig.v1.OsConfigService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.osconfig.v1.OsConfigZonalService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}

	if strings.Contains(path, "/patchJobs") ||
		strings.Contains(path, "/patchDeployments") ||
		strings.Contains(path, ":pause") ||
		strings.Contains(path, ":resume") {
		return true
	}

	if strings.Contains(path, "/osPolicyAssignments") ||
		strings.Contains(path, ":listRevisions") {
		return true
	}

	if strings.Contains(path, "/instances/") &&
		(strings.Contains(path, "/inventories") || strings.Contains(path, "/vulnerabilityReports")) {
		return true
	}

	return strings.Contains(path, "/osPolicyAssignments/") && strings.Contains(path, "/reports")
}

func handleGCPContractProbe_osconfig(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "osconfig") {
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
			"name":     "projects/stackyard/locations/us-central1/osconfig/sample",
			"service":  "osconfig",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
