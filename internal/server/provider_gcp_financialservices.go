package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPFinancialServicesRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_financialservices(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPFinancialServicesPath(path) {
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

func isGCPFinancialServicesPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.financialservices.v1.AML/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if isGCPFinancialServicesLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/instances") ||
		strings.Contains(path, "/datasets") ||
		strings.Contains(path, "/models") ||
		strings.Contains(path, "/engineConfigs") ||
		strings.Contains(path, "/engineVersions") ||
		strings.Contains(path, "/predictionResults") ||
		strings.Contains(path, "/backtestResults") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":importRegisteredParties") ||
		strings.Contains(path, ":exportRegisteredParties") ||
		strings.Contains(path, ":exportMetadata") ||
		strings.Contains(path, ":cancel")
}

func isGCPFinancialServicesLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_financialservices(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "financialservices") {
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
			"name":     "projects/stackyard/locations/us-central1/financialservices/sample",
			"service":  "financialservices",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
