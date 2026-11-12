package server

import (
	"net/http"
	"strings"
)

const azureFleet20250301APIVersion = "2025-03-01"

// handleAzureFleet20250301Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/fleet-2025-03-01-plan.md.
func (s *Server) handleAzureFleet20250301Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsFleet20250301Path(segments, r.URL.Query().Get("api-version")) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if !azureFleet20250301AllowedMethod(r.Method) {
		return false
	}

	respondAzureImplemented(w, path)
	return true
}

func azureIsFleet20250301Path(segments []string, apiVersion string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ContainerService") &&
		strings.EqualFold(segments[2], "operations") {
		return strings.EqualFold(strings.TrimSpace(apiVersion), azureFleet20250301APIVersion)
	}

	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return false
	}

	for idx := 2; idx+2 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") &&
			strings.EqualFold(segments[idx+1], "Microsoft.ContainerService") &&
			strings.EqualFold(segments[idx+2], "fleets") {
			return true
		}
	}

	return false
}

func azureFleet20250301AllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
