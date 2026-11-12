package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleAzureAppConfigurationResourceManagerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)

	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 {
		return false
	}
	if !azureIsAppConfigurationResourceManagerPath(segments) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if !azureAppConfigurationResourceManagerAllowedMethod(r.Method) {
		return false
	}

	respondAzureImplemented(w, path)
	return true
}

func azureIsAppConfigurationResourceManagerPath(segments []string) bool {
	// Provider-level endpoints:
	// /providers/Microsoft.AppConfiguration/operations
	if len(segments) >= 2 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AppConfiguration") {
		return true
	}

	// Subscription/resource-group scoped ARM paths:
	// /subscriptions/{id}/.../providers/Microsoft.AppConfiguration/...
	if len(segments) < 2 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.AppConfiguration") {
			return true
		}
	}
	return false
}

func azureAppConfigurationResourceManagerAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
