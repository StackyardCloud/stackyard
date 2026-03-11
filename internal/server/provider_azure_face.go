package server

import (
	"net/http"
	"strings"
)

const azureFacePrefix = "/azure/face/v1.2/"

func (s *Server) handleAzureFaceRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureFacePrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureFacePrefix)
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
	case "detect", "findsimilars", "group", "identify", "verify":
		if r.Method == http.MethodPost {
			respondAzureImplemented(w, path)
			return true
		}
	case "facelists", "largefacelists", "persongroups", "largepersongroups":
		if azureFaceResourceAllowsMethod(r.Method) {
			respondAzureImplemented(w, path)
			return true
		}
	case "detectliveness-sessions", "detectlivenesswithverify-sessions":
		if (len(segments) == 1 && r.Method == http.MethodPost) || (len(segments) >= 2 && (r.Method == http.MethodGet || r.Method == http.MethodDelete)) {
			respondAzureImplemented(w, path)
			return true
		}
	case "sessionimages":
		if len(segments) >= 2 && r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	}

	// Keep staged ownership for unknown routes under the face prefix.
	respondAzureImplemented(w, path)
	return true
}

func azureFaceResourceAllowsMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
