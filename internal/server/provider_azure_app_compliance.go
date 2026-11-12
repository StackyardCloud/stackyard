package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleAzureAppComplianceRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)

	if !strings.HasPrefix(path, "/azure/providers/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) < 2 || !strings.EqualFold(segments[0], "providers") || !strings.EqualFold(segments[1], "Microsoft.AppComplianceAutomation") {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if !azureAppComplianceAllowedMethod(r.Method) {
		return false
	}

	respondAzureImplemented(w, path)
	return true
}

func azureAppComplianceAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
