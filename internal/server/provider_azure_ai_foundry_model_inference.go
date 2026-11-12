package server

import (
	"net/http"
	"strings"
)

const azureAIFoundryModelInferencePrefix = "/azure/ai-foundry/model-inference/"

func (s *Server) handleAzureAIFoundryModelInferenceRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureAIFoundryModelInferencePrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureAIFoundryModelInferencePrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) == 0 {
		return false
	}

	if azureAIFoundryModelInferenceKnownRoute(segments, r.Method) {
		respondAzureImplemented(w, path)
		return true
	}

	// Keep staged ownership for the full model-inference prefix.
	respondAzureImplemented(w, path)
	return true
}

func azureAIFoundryModelInferenceKnownRoute(segments []string, method string) bool {
	switch {
	case len(segments) == 3 &&
		strings.EqualFold(segments[0], "models") &&
		strings.EqualFold(segments[1], "chat") &&
		strings.EqualFold(segments[2], "completions") &&
		method == http.MethodPost:
		return true
	case len(segments) == 2 &&
		strings.EqualFold(segments[0], "models") &&
		strings.EqualFold(segments[1], "embeddings") &&
		method == http.MethodPost:
		return true
	case len(segments) == 3 &&
		strings.EqualFold(segments[0], "models") &&
		strings.EqualFold(segments[1], "images") &&
		strings.EqualFold(segments[2], "embeddings") &&
		method == http.MethodPost:
		return true
	case len(segments) == 2 &&
		strings.EqualFold(segments[0], "models") &&
		strings.EqualFold(segments[1], "info") &&
		method == http.MethodGet:
		return true
	default:
		return false
	}
}
