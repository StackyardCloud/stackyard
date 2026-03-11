package server

import (
	"net/http"
	"strings"
)

const azureCustomVoicePrefix = "/azure/customvoice/"

func (s *Server) handleAzureCustomVoiceRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCustomVoicePrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureCustomVoicePrefix)
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
	case "basemodels":
		if len(segments) == 1 && r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	case "consents", "endpoints", "models", "personalvoices", "projects", "trainingsets":
		if azureCustomVoiceResourceAllowsMethod(r.Method) {
			respondAzureImplemented(w, path)
			return true
		}
	case "modelrecipes":
		if len(segments) == 1 && r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	case "operations":
		if len(segments) >= 2 && strings.TrimSpace(segments[1]) != "" && r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	}

	// Keep staged ownership for unknown custom voice routes under the full prefix.
	respondAzureImplemented(w, path)
	return true
}

func azureCustomVoiceResourceAllowsMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
