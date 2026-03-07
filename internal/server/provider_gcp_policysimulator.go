package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPPolicySimulatorRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_policysimulator(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPPolicySimulatorPath(path) {
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

func isGCPPolicySimulatorPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.policysimulator.v1.Simulator/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.policysimulator.v1.OrgPolicyViolationsPreviewService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/") {
		return false
	}

	return strings.Contains(path, "/replays") ||
		strings.Contains(path, "/orgPolicyViolationsPreviews") ||
		strings.Contains(path, "/orgPolicyViolations")
}

func handleGCPContractProbe_policysimulator(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "policysimulator") {
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
			"name":     "projects/stackyard/locations/us-central1/policysimulator/sample",
			"service":  "policysimulator",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
