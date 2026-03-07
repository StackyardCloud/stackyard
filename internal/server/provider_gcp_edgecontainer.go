package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPEdgeContainerRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_edgecontainer(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPEdgeContainerPath(path) {
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

func isGCPEdgeContainerPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.edgecontainer.v1.EdgeContainer/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/clusters") ||
		strings.Contains(path, "/nodePools") ||
		strings.Contains(path, "/machines") ||
		strings.Contains(path, "/vpnConnections") ||
		strings.Contains(path, "/serverConfig") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":generateAccessToken") ||
		strings.Contains(path, ":generateOfflineCredential") ||
		strings.Contains(path, ":upgrade") ||
		strings.Contains(path, ":cancel")
}

func handleGCPContractProbe_edgecontainer(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "edgecontainer") {
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
			"name":     "projects/stackyard/locations/us-central1/edgecontainer/sample",
			"service":  "edgecontainer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
