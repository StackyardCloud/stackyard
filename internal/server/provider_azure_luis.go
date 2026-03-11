package server

import (
	"net/http"
	"strings"
)

const azureLuisPrefix = "/azure/luis/prediction/v3.0/"

func (s *Server) handleAzureLuisRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureLuisPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureLuisPrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) < 5 {
		respondAzureImplemented(w, path)
		return true
	}

	// LUIS Prediction v3.0 documented envelopes:
	// - /apps/{appId}/slots/{slotName}/predict (GET/POST)
	// - /apps/{appId}/versions/{versionId}/predict (GET/POST)
	if strings.EqualFold(segments[0], "apps") &&
		strings.TrimSpace(segments[1]) != "" &&
		(strings.EqualFold(segments[2], "slots") || strings.EqualFold(segments[2], "versions")) &&
		strings.TrimSpace(segments[3]) != "" &&
		strings.EqualFold(segments[4], "predict") &&
		len(segments) == 5 {
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondAzureImplemented(w, path)
			return true
		}
	}

	// Keep staged ownership for the full LUIS prediction prefix.
	respondAzureImplemented(w, path)
	return true
}
