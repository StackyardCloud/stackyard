package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPKMSInventoryRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_kms_inventory(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPKMSInventoryPath(path) {
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

func isGCPKMSInventoryPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.kms.inventory.v1.KeyDashboardService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.kms.inventory.v1.KeyTrackingService/") {
		return true
	}

	if strings.HasPrefix(path, "/gcp/v1/projects/") &&
		strings.HasSuffix(path, "/cryptoKeys") &&
		!strings.Contains(path, "/locations/") {
		return true
	}

	if strings.HasPrefix(path, "/gcp/v1/") &&
		strings.Contains(path, "/cryptoKeys/") &&
		strings.HasSuffix(path, "/protectedResourcesSummary") {
		return true
	}

	return strings.HasPrefix(path, "/gcp/v1/organizations/") &&
		strings.HasSuffix(path, "/protectedResources:search")
}

func handleGCPContractProbe_kms_inventory(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "kms_inventory") {
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
			"name":     "projects/stackyard/locations/us-central1/kms_inventory/sample",
			"service":  "kms_inventory",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
