package server

import (
	"net/http"
	"strings"
)

const azureTranslatorPrefix = "/azure/translator/"

func (s *Server) handleAzureTranslatorRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureTranslatorPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureTranslatorPrefix)
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
	case "translate", "detect", "breaksentence":
		if len(segments) == 1 && r.Method == http.MethodPost {
			respondAzureImplemented(w, path)
			return true
		}
	case "languages":
		if len(segments) == 1 && r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	case "dictionary":
		if len(segments) == 2 && r.Method == http.MethodPost {
			action := strings.ToLower(strings.TrimSpace(segments[1]))
			if action == "lookup" || action == "examples" {
				if !azureTranslatorRequireQueryParam(w, r, path, "from") {
					return true
				}
				if !azureTranslatorRequireQueryParam(w, r, path, "to") {
					return true
				}
				respondAzureImplemented(w, path)
				return true
			}
		}
	case "transliterate":
		if len(segments) == 1 && r.Method == http.MethodPost {
			if !azureTranslatorRequireQueryParam(w, r, path, "language") {
				return true
			}
			if !azureTranslatorRequireQueryParam(w, r, path, "fromScript") {
				return true
			}
			if !azureTranslatorRequireQueryParam(w, r, path, "toScript") {
				return true
			}
			respondAzureImplemented(w, path)
			return true
		}
	}

	// Keep staged ownership for unknown nested routes under this prefix.
	respondAzureImplemented(w, path)
	return true
}

func azureTranslatorRequireQueryParam(w http.ResponseWriter, r *http.Request, path, key string) bool {
	if strings.TrimSpace(r.URL.Query().Get(key)) == "" {
		respondAzureInvalidRequest(w, path, key+" query parameter is required")
		return false
	}
	return true
}
