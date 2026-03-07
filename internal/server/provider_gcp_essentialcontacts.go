package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPEssentialContactsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_essentialcontacts(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPEssentialContactsPath(path) {
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

func isGCPEssentialContactsPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.essentialcontacts.v1.EssentialContactsService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/") {
		return false
	}
	if !strings.Contains(path, "/contacts") {
		return false
	}

	return strings.Contains(path, "/projects/") ||
		strings.Contains(path, "/folders/") ||
		strings.Contains(path, "/organizations/")
}

func handleGCPContractProbe_essentialcontacts(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "essentialcontacts") {
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
			"name":     "projects/stackyard/locations/us-central1/essentialcontacts/sample",
			"service":  "essentialcontacts",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
