package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleAzureAKSRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)

	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 {
		return false
	}
	if !azureIsAKSPath(segments) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if !azureAKSAllowedMethod(r.Method) {
		return false
	}

	respondAzureImplemented(w, path)
	return true
}

func azureIsAKSPath(segments []string) bool {
	// Provider-level operations:
	// /providers/Microsoft.ContainerService/operations
	if len(segments) >= 2 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ContainerService") {
		return true
	}

	// Subscription/resource-group scoped ARM paths:
	// /subscriptions/{id}/.../providers/Microsoft.ContainerService/...
	if len(segments) < 2 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.ContainerService") {
			return true
		}
	}
	return false
}

func azureAKSAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
