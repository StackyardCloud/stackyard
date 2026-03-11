package server

import (
	"net/http"
	"strings"
)

const azureVideoTranslationPrefix = "/azure/videotranslation/"

func (s *Server) handleAzureVideoTranslationRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureVideoTranslationPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureVideoTranslationPrefix)
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
	case "configurations":
		if handleAzureVideoTranslationConfigurationRoutes(w, r, path, segments) {
			return true
		}
	case "translations":
		if handleAzureVideoTranslationTranslationRoutes(w, r, path, segments) {
			return true
		}
	case "operations":
		if handleAzureVideoTranslationOperationRoutes(w, r, path, segments) {
			return true
		}
	}

	// Keep staged ownership for unknown routes under this prefix.
	respondAzureImplemented(w, path)
	return true
}

func handleAzureVideoTranslationConfigurationRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	if len(segments) != 2 {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(segments[1])) {
	case "event-hub":
		switch r.Method {
		case http.MethodPut, http.MethodGet, http.MethodDelete:
			respondAzureImplemented(w, path)
			return true
		}
	case "event-hub:ping":
		if r.Method == http.MethodPost {
			respondAzureImplemented(w, path)
			return true
		}
	}

	return false
}

func handleAzureVideoTranslationTranslationRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	switch len(segments) {
	case 1:
		if r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
		if r.Method == http.MethodPut || r.Method == http.MethodDelete {
			respondAzureInvalidRequest(w, path, "translationId path parameter is required")
			return true
		}
	case 2:
		if strings.TrimSpace(segments[1]) == "" {
			respondAzureInvalidRequest(w, path, "translationId path parameter is required")
			return true
		}
		if r.Method == http.MethodPut || r.Method == http.MethodGet || r.Method == http.MethodDelete {
			respondAzureImplemented(w, path)
			return true
		}
	case 3:
		if !strings.EqualFold(segments[2], "iterations") {
			return false
		}
		if strings.TrimSpace(segments[1]) == "" {
			respondAzureInvalidRequest(w, path, "translationId path parameter is required")
			return true
		}
		switch r.Method {
		case http.MethodGet:
			respondAzureImplemented(w, path)
			return true
		case http.MethodPut:
			respondAzureInvalidRequest(w, path, "iterationId path parameter is required")
			return true
		}
	case 4:
		if !strings.EqualFold(segments[2], "iterations") {
			return false
		}
		if strings.TrimSpace(segments[1]) == "" {
			respondAzureInvalidRequest(w, path, "translationId path parameter is required")
			return true
		}
		if strings.TrimSpace(segments[3]) == "" {
			respondAzureInvalidRequest(w, path, "iterationId path parameter is required")
			return true
		}
		if r.Method == http.MethodPut || r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	}

	return false
}

func handleAzureVideoTranslationOperationRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	if r.Method != http.MethodGet {
		return false
	}

	switch len(segments) {
	case 1:
		respondAzureInvalidRequest(w, path, "operationId path parameter is required")
		return true
	case 2:
		if strings.TrimSpace(segments[1]) == "" {
			respondAzureInvalidRequest(w, path, "operationId path parameter is required")
			return true
		}
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}
