package server

import (
	"net/http"
	"strings"
)

const azureBatchTextToSpeechPrefix = "/azure/batchtexttospeech/2024-04-01/"

func (s *Server) handleAzureTextToSpeechRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureBatchTextToSpeechPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureBatchTextToSpeechPrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) == 0 {
		return false
	}

	resource := strings.ToLower(strings.TrimSpace(segments[0]))
	switch resource {
	case "batchsyntheses":
		switch {
		case len(segments) == 1 && r.Method == http.MethodGet:
			respondAzureImplemented(w, path)
			return true
		case len(segments) == 2 && segments[1] != "" && (r.Method == http.MethodPut || r.Method == http.MethodGet || r.Method == http.MethodDelete):
			respondAzureImplemented(w, path)
			return true
		default:
			respondAzureImplemented(w, path)
			return true
		}
	case "operations":
		if len(segments) == 2 && segments[1] != "" && r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
		respondAzureImplemented(w, path)
		return true
	default:
		// Keep staged ownership for the full batch text-to-speech prefix.
		respondAzureImplemented(w, path)
		return true
	}
}
