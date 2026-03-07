package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPLocationFinderRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_locationfinder(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPLocationFinderPath(path) {
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

func isGCPLocationFinderPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.locationfinder.v1.CloudLocationFinder/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if isGCPLocationFinderLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/cloudLocations") ||
		strings.Contains(path, "/cloudLocations:search")
}

func isGCPLocationFinderLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_locationfinder(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "locationfinder") {
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
			"name":     "projects/stackyard/locations/us-central1/locationfinder/sample",
			"service":  "locationfinder",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
