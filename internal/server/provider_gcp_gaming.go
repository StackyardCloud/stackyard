package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPGameServicesRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_gaming(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPGameServicesPath(path) {
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

func isGCPGameServicesPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.gaming.v1.RealmsService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.gaming.v1.GameServerClustersService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.gaming.v1.GameServerConfigsService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.gaming.v1.GameServerDeploymentsService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/realms") ||
		strings.Contains(path, "/gameServerClusters") ||
		strings.Contains(path, "/gameServerDeployments") ||
		strings.Contains(path, "/configs") ||
		strings.Contains(path, "/rollout") ||
		strings.Contains(path, ":previewCreate") ||
		strings.Contains(path, ":previewDelete") ||
		strings.Contains(path, ":previewUpdate") ||
		strings.Contains(path, ":preview") ||
		strings.Contains(path, ":fetchDeploymentState") ||
		strings.Contains(path, "/operations")
}

func handleGCPContractProbe_gaming(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "gaming") {
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
			"name":     "projects/stackyard/locations/us-central1/gaming/sample",
			"service":  "gaming",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
