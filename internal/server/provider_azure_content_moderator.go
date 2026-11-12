package server

import (
	"net/http"
	"strings"
)

const azureContentModeratorPrefix = "/azure/contentmoderator/"

func (s *Server) handleAzureContentModeratorRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureContentModeratorPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureContentModeratorPrefix)
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
	case "moderate", "lists", "review", "reviews":
		respondAzureImplemented(w, path)
		return true
	default:
		// Keep staged ownership for unknown paths under the full prefix.
		respondAzureImplemented(w, path)
		return true
	}
}
