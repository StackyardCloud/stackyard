package server

import (
	"net/http"
	"strings"
)

const azureHealthInsightsPrefix = "/azure/health-insights/"

func (s *Server) handleAzureHealthInsightsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureHealthInsightsPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureHealthInsightsPrefix)
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
	case "radiology-insights":
		if len(segments) >= 2 && strings.EqualFold(segments[1], "jobs") {
			if len(segments) >= 3 && strings.TrimSpace(segments[2]) != "" {
				if r.Method == http.MethodPut || r.Method == http.MethodGet {
					respondAzureImplemented(w, path)
					return true
				}
			}
			respondAzureImplemented(w, path)
			return true
		}
	}

	// Keep staged ownership for unknown routes under the health-insights prefix.
	respondAzureImplemented(w, path)
	return true
}
