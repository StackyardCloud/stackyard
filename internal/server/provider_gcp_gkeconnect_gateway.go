package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPGKEConnectGatewayRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_gkeconnect_gateway(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPGKEConnectGatewayPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPGKEConnectGatewayPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.gkeconnect.gateway.v1.GatewayControl/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") || !strings.Contains(path, "/memberships/") {
		return false
	}
	return strings.Contains(path, ":generateCredentials")
}

func handleGCPContractProbe_gkeconnect_gateway(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "gkeconnect_gateway") {
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
			"name":     "projects/stackyard/locations/us-central1/gkeconnect_gateway/sample",
			"service":  "gkeconnect_gateway",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
