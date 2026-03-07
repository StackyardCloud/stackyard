package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPLicenseManagerRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_licensemanager(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPLicenseManagerPath(path) {
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

func isGCPLicenseManagerPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.licensemanager.v1.LicenseManager/") {
		return true
	}

	if !strings.HasPrefix(path, "/gcp/v1/projects/") || !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/configurations") ||
		strings.Contains(path, "/instances") ||
		strings.Contains(path, "/products") ||
		strings.Contains(path, ":deactivate") ||
		strings.Contains(path, ":reactivate") ||
		strings.Contains(path, ":queryLicenseUsage") ||
		strings.Contains(path, ":aggregateUsage")
}

func handleGCPContractProbe_licensemanager(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "licensemanager") {
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
			"name":     "projects/stackyard/locations/us-central1/licensemanager/sample",
			"service":  "licensemanager",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
