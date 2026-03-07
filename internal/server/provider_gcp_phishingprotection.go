package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPPhishingProtectionRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_phishingprotection(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPPhishingProtectionPath(path) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func isGCPPhishingProtectionPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.phishingprotection.v1beta1.PhishingProtectionServiceV1Beta1/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1beta1/projects/") {
		return false
	}
	return strings.Contains(path, "/phishing:report")
}

func handleGCPContractProbe_phishingprotection(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "phishingprotection") {
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
			"name":     "projects/stackyard/locations/us-central1/phishingprotection/sample",
			"service":  "phishingprotection",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
