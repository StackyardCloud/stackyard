package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPOsConfigAgentEndpointRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_osconfig_agentendpoint(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPOsConfigAgentEndpointPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodPost:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPOsConfigAgentEndpointPath(path string) bool {
	return strings.HasPrefix(path, "/gcp/google.cloud.osconfig.agentendpoint.v1.AgentEndpointService/")
}

func handleGCPContractProbe_osconfig_agentendpoint(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "osconfig_agentendpoint") {
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
			"name":     "projects/stackyard/locations/us-central1/osconfig_agentendpoint/sample",
			"service":  "osconfig_agentendpoint",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
