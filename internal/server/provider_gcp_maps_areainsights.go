package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPMapsAreaInsightsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_maps_areainsights(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPMapsAreaInsightsPath(path) {
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

func isGCPMapsAreaInsightsPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.maps.areainsights.v1.AreaInsights/") {
		return true
	}

	return path == "/gcp/v1:computeInsights"
}

func handleGCPContractProbe_maps_areainsights(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "maps_areainsights") {
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
			"name":     "projects/stackyard/locations/us-central1/maps_areainsights/sample",
			"service":  "maps_areainsights",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
