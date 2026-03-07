package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPEdgeNetworkRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_edgenetwork(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPEdgeNetworkPath(path) {
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

func isGCPEdgeNetworkPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.edgenetwork.v1.EdgeNetwork/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/zones") ||
		strings.Contains(path, "/networks") ||
		strings.Contains(path, "/subnets") ||
		strings.Contains(path, "/interconnects") ||
		strings.Contains(path, "/interconnectAttachments") ||
		strings.Contains(path, "/routers") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":initialize") ||
		strings.Contains(path, ":diagnose") ||
		strings.Contains(path, ":cancel")
}

func handleGCPContractProbe_edgenetwork(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "edgenetwork") {
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
			"name":     "projects/stackyard/locations/us-central1/edgenetwork/sample",
			"service":  "edgenetwork",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
