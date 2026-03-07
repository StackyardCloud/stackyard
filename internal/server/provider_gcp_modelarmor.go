package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPModelArmorRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_modelarmor(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPModelArmorPath(path) {
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

func isGCPModelArmorPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.modelarmor.v1.ModelArmor/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}

	return strings.Contains(path, "/templates") ||
		strings.Contains(path, "/floorSetting") ||
		strings.Contains(path, "/floorSettings") ||
		strings.Contains(path, "/floorsetting") ||
		strings.Contains(path, ":sanitizeUserPrompt") ||
		strings.Contains(path, ":sanitizeModelResponse")
}

func handleGCPContractProbe_modelarmor(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "modelarmor") {
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
			"name":     "projects/stackyard/locations/us-central1/modelarmor/sample",
			"service":  "modelarmor",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
