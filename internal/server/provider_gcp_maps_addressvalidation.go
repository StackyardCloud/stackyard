package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPMapsAddressValidationRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_maps_addressvalidation(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPMapsAddressValidationPath(path) {
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

func isGCPMapsAddressValidationPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.maps.addressvalidation.v1.AddressValidation/") {
		return true
	}

	return path == "/gcp/v1:validateAddress" ||
		path == "/gcp/v1:provideValidationFeedback"
}

func handleGCPContractProbe_maps_addressvalidation(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "maps_addressvalidation") {
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
			"name":     "projects/stackyard/locations/us-central1/maps_addressvalidation/sample",
			"service":  "maps_addressvalidation",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
