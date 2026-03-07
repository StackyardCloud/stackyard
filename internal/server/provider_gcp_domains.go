package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDomainsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_domains(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPDomainsPath(path) {
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

func isGCPDomainsPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.domains.v1beta1.Domains/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1beta1/") {
		return false
	}

	return strings.Contains(path, "/registrations") ||
		strings.Contains(path, "/operations/") ||
		strings.Contains(path, ":searchDomains") ||
		strings.Contains(path, ":retrieveRegisterParameters") ||
		strings.Contains(path, ":register") ||
		strings.Contains(path, ":retrieveTransferParameters") ||
		strings.Contains(path, ":transfer") ||
		strings.Contains(path, ":configureManagementSettings") ||
		strings.Contains(path, ":configureDnsSettings") ||
		strings.Contains(path, ":configureContactSettings") ||
		strings.Contains(path, ":export") ||
		strings.Contains(path, ":retrieveAuthorizationCode") ||
		strings.Contains(path, ":resetAuthorizationCode") ||
		strings.Contains(path, ":cancel")
}

func handleGCPContractProbe_domains(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "domains") {
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
			"name":     "projects/stackyard/locations/us-central1/domains/sample",
			"service":  "domains",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
