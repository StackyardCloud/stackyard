package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDeviceStreamingRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_devicestreaming(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPDeviceStreamingPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPatch:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPDeviceStreamingPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/v1/projects/") && strings.Contains(path, "/deviceSessions") {
		return true
	}
	return strings.HasPrefix(path, "/gcp/google.cloud.devicestreaming.v1.DirectAccessService/")
}

func handleGCPContractProbe_devicestreaming(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "devicestreaming") {
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
			"name":     "projects/stackyard/locations/us-central1/devicestreaming/sample",
			"service":  "devicestreaming",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
