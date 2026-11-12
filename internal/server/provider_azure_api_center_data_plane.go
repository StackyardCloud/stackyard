package server

import (
	"net/http"
	"strings"
)

const (
	azureAPICenterPrefix    = "/azure/apicenter/"
	azureAPICenterAltPrefix = "/azure/api-center/"
)

func (s *Server) handleAzureAPICenterDataPlaneRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !azureIsAPICenterDataPlanePath(path) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if !azureAPICenterDataPlaneAllowedMethod(r.Method) {
		return false
	}

	respondAzureImplemented(w, path)
	return true
}

func azureIsAPICenterDataPlanePath(path string) bool {
	relative, ok := azureTrimAnyPrefix(path, azureAPICenterPrefix, azureAPICenterAltPrefix)
	if !ok {
		return false
	}
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) == 0 {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(segments[0])) {
	case "workspaces", "apis", "environments":
		return true
	default:
		return false
	}
}

func azureAPICenterDataPlaneAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost:
		return true
	default:
		return false
	}
}

func azureTrimAnyPrefix(path string, prefixes ...string) (string, bool) {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return strings.TrimPrefix(path, prefix), true
		}
	}
	return "", false
}
