package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPIoTRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_iot(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPIoTPath(path) {
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

func isGCPIoTPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.iot.v1.DeviceManager/") {
		return true
	}

	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/registries")
}

func handleGCPContractProbe_iot(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "iot") {
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
			"name":     "projects/stackyard/locations/us-central1/iot/sample",
			"service":  "iot",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
