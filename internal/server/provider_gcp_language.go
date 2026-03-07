package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPLanguageRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_language(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPLanguagePath(path) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func isGCPLanguagePath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.language.v1.LanguageService/") {
		return true
	}

	switch path {
	case "/gcp/v1/documents:analyzeSentiment",
		"/gcp/v1/documents:analyzeEntities",
		"/gcp/v1/documents:analyzeEntitySentiment",
		"/gcp/v1/documents:analyzeSyntax",
		"/gcp/v1/documents:classifyText",
		"/gcp/v1/documents:moderateText",
		"/gcp/v1/documents:annotateText":
		return true
	default:
		return false
	}
}

func handleGCPContractProbe_language(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "language") {
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
			"name":     "projects/stackyard/locations/us-central1/language/sample",
			"service":  "language",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
