package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPPolicyTroubleshooterRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_policytroubleshooter(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPPolicyTroubleshooterPath(path) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func isGCPPolicyTroubleshooterPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.policytroubleshooter.v1.IamChecker/") {
		return true
	}
	return strings.HasPrefix(path, "/gcp/v1/iam:troubleshoot")
}

func handleGCPContractProbe_policytroubleshooter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "policytroubleshooter") {
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
			"name":     "projects/stackyard/locations/us-central1/policytroubleshooter/sample",
			"service":  "policytroubleshooter",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
