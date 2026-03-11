package server

import (
	"net/http"
	"strings"
)

const azurePersonalizerPrefix = "/azure/personalizer/v1.0/"

func (s *Server) handleAzurePersonalizerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azurePersonalizerPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azurePersonalizerPrefix)
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
	case "rank":
		if len(segments) == 1 && r.Method == http.MethodPost {
			respondAzureImplemented(w, path)
			return true
		}
	case "events":
		// Documented:
		// POST /personalizer/v1.0/events/{eventId}/activate
		// POST /personalizer/v1.0/events/{eventId}/reward
		if len(segments) == 3 && strings.TrimSpace(segments[1]) != "" && r.Method == http.MethodPost {
			action := strings.ToLower(strings.TrimSpace(segments[2]))
			if action == "activate" || action == "reward" {
				respondAzureImplemented(w, path)
				return true
			}
		}
	}

	// Keep staged ownership for unknown routes under the personalizer prefix.
	respondAzureImplemented(w, path)
	return true
}
