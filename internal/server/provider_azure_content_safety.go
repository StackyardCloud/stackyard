package server

import (
	"net/http"
	"strings"
)

const azureContentSafetyPrefix = "/azure/contentsafety/"

func (s *Server) handleAzureContentSafetyRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureContentSafetyPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureContentSafetyPrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) == 0 {
		return false
	}

	if handleAzureContentSafetyImageRoutes(w, r, path, segments) {
		return true
	}
	if handleAzureContentSafetyTextRoutes(w, r, path, segments) {
		return true
	}

	// Keep staged ownership for the full content safety prefix.
	respondAzureImplemented(w, path)
	return true
}

func handleAzureContentSafetyImageRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	if len(segments) != 1 || r.Method != http.MethodPost {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(segments[0])) {
	case "image:analyze":
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureContentSafetyTextRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	if len(segments) == 1 && r.Method == http.MethodPost {
		switch strings.ToLower(strings.TrimSpace(segments[0])) {
		case "text:analyze", "text:detectprotectedmaterial", "text:shieldprompt":
			respondAzureImplemented(w, path)
			return true
		}
		return false
	}

	if len(segments) < 2 {
		return false
	}
	if !strings.EqualFold(segments[0], "text") || !strings.EqualFold(segments[1], "blocklists") {
		return false
	}

	switch {
	case len(segments) == 2 && r.Method == http.MethodGet:
		// List Text Blocklists
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 3 && segments[2] != "":
		segment := strings.TrimSpace(segments[2])
		lower := strings.ToLower(segment)
		if strings.Contains(lower, ":") {
			switch {
			case strings.HasSuffix(lower, ":addorupdateblocklistitems") && r.Method == http.MethodPost:
				respondAzureImplemented(w, path)
				return true
			case strings.HasSuffix(lower, ":removeblocklistitems") && r.Method == http.MethodPost:
				respondAzureImplemented(w, path)
				return true
			default:
				return false
			}
		}
		switch {
		case r.Method == http.MethodPatch || r.Method == http.MethodDelete || r.Method == http.MethodGet:
			respondAzureImplemented(w, path)
			return true
		default:
			return false
		}
	case len(segments) == 4 && segments[2] != "" && strings.EqualFold(segments[3], "blocklistItems") && r.Method == http.MethodGet:
		// List Text Blocklist Items
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 5 && segments[2] != "" && strings.EqualFold(segments[3], "blocklistItems") && segments[4] != "" && r.Method == http.MethodGet:
		// Get Text Blocklist Item
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}
