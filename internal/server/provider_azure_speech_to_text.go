package server

import (
	"net/http"
	"strings"
)

const azureSpeechToTextPrefix = "/azure/speechtotext/v3.2-preview.2/"

func (s *Server) handleAzureSpeechToTextRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSpeechToTextPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureSpeechToTextPrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) == 0 {
		return false
	}
	resource := strings.ToLower(strings.TrimSpace(segments[0]))
	if resource == "" {
		return false
	}

	switch resource {
	case "healthstatus", "servicehealth":
		if len(segments) == 1 && r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	case "operations":
		if r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	}

	if azureSpeechToTextKnownResource(resource) && azureSpeechToTextResourceAllowsMethod(r.Method) {
		respondAzureImplemented(w, path)
		return true
	}

	// Keep staged ownership for the entire speech-to-text prefix to avoid
	// cross-routing into unrelated Azure handlers.
	respondAzureImplemented(w, path)
	return true
}

func azureSpeechToTextKnownResource(resource string) bool {
	switch resource {
	case "datasets", "endpoints", "evaluations", "models", "projects", "transcriptions", "webhooks":
		return true
	default:
		return false
	}
}

func azureSpeechToTextResourceAllowsMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
