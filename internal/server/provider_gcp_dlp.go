package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDLPRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_dlp(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPDLPPath(path) {
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

func isGCPDLPPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.privacy.dlp.v2.DlpService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v2/") {
		return false
	}

	return strings.Contains(path, "/infoTypes") ||
		strings.Contains(path, "/inspectTemplates") ||
		strings.Contains(path, "/deidentifyTemplates") ||
		strings.Contains(path, "/jobTriggers") ||
		strings.Contains(path, "/discoveryConfigs") ||
		strings.Contains(path, "/dlpJobs") ||
		strings.Contains(path, "/storedInfoTypes") ||
		strings.Contains(path, "/projectDataProfiles") ||
		strings.Contains(path, "/tableDataProfiles") ||
		strings.Contains(path, "/columnDataProfiles") ||
		strings.Contains(path, "/fileStoreDataProfiles") ||
		strings.Contains(path, "/connections") ||
		strings.Contains(path, "content:inspect") ||
		strings.Contains(path, "content:deidentify") ||
		strings.Contains(path, "content:reidentify") ||
		strings.Contains(path, "image:redact") ||
		strings.Contains(path, ":hybridInspect") ||
		strings.Contains(path, ":activate") ||
		strings.Contains(path, ":cancel") ||
		strings.Contains(path, ":finish") ||
		strings.Contains(path, "connections:search")
}

func handleGCPContractProbe_dlp(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "dlp") {
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
			"name":     "projects/stackyard/locations/us-central1/dlp/sample",
			"service":  "dlp",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
