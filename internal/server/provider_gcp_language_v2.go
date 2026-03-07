package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPLanguageV2Router(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_language_v2(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPLanguageV2Path(path) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func isGCPLanguageV2Path(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.language.v2.LanguageService/") {
		return true
	}

	switch path {
	case "/gcp/v2/documents:analyzeSentiment",
		"/gcp/v2/documents:analyzeEntities",
		"/gcp/v2/documents:classifyText",
		"/gcp/v2/documents:moderateText",
		"/gcp/v2/documents:annotateText":
		return true
	default:
		return false
	}
}

func handleGCPContractProbe_language_v2(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "language_v2") {
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
			"name":     "projects/stackyard/locations/us-central1/language_v2/sample",
			"service":  "language_v2",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
