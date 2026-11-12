package server

import (
	"net/http"
	"strings"
)

const azureCustomVisionPrefix = "/azure/customvision/v3.3/"

func (s *Server) handleAzureCustomVisionRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCustomVisionPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureCustomVisionPrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) == 0 {
		return false
	}

	if strings.EqualFold(strings.TrimSpace(segments[0]), "training") {
		if handleAzureCustomVisionTrainingRoutes(w, r, path, segments) {
			return true
		}
	}

	// Keep staged ownership for unknown routes under the full custom-vision prefix.
	respondAzureImplemented(w, path)
	return true
}

func handleAzureCustomVisionTrainingRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	if len(segments) < 2 {
		return false
	}

	resource := strings.ToLower(strings.TrimSpace(segments[1]))
	switch resource {
	case "domains":
		return handleAzureCustomVisionDomainsRoutes(w, r, path, segments[1:])
	case "projects":
		return handleAzureCustomVisionProjectsRoutes(w, r, path, segments[1:])
	default:
		return false
	}
}

func handleAzureCustomVisionDomainsRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	switch {
	case len(segments) == 1 && r.Method == http.MethodGet:
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && strings.TrimSpace(segments[1]) != "" && r.Method == http.MethodGet:
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureCustomVisionProjectsRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	if len(segments) == 0 || !strings.EqualFold(strings.TrimSpace(segments[0]), "projects") {
		return false
	}
	if !azureCustomVisionSupportsMethod(r.Method) {
		return false
	}

	switch {
	case len(segments) == 1 && (r.Method == http.MethodGet || r.Method == http.MethodPost):
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && strings.EqualFold(segments[1], "import") && r.Method == http.MethodPost:
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && strings.TrimSpace(segments[1]) != "" && azureCustomVisionProjectAllowsMethod(r.Method):
		respondAzureImplemented(w, path)
		return true
	case len(segments) >= 3 && strings.TrimSpace(segments[1]) != "":
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func azureCustomVisionProjectAllowsMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func azureCustomVisionSupportsMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
