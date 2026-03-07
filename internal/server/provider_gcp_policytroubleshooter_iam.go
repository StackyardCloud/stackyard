package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPPolicyTroubleshooterIAMRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_policytroubleshooter_iam(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPPolicyTroubleshooterIAMPath(path) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func isGCPPolicyTroubleshooterIAMPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.policytroubleshooter.iam.v3.PolicyTroubleshooter/") {
		return true
	}
	return strings.HasPrefix(path, "/gcp/v3/iam:troubleshoot")
}

func handleGCPContractProbe_policytroubleshooter_iam(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "policytroubleshooter_iam") {
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
			"name":     "projects/stackyard/locations/us-central1/policytroubleshooter_iam/sample",
			"service":  "policytroubleshooter_iam",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
