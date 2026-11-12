package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleAzureAPIManagementResourceManagerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)

	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 {
		return false
	}
	if !azureIsAPIManagementResourceManagerPath(segments) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if !azureAPIManagementResourceManagerAllowedMethod(r.Method) {
		return false
	}

	if r.Method == http.MethodHead && azureAPIManagementIsCheckEntityExists(segments) {
		w.WriteHeader(http.StatusNoContent)
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

// azureAPIManagementCheckEntityExistsRoutes are the service-relative routes whose
// HEAD (CheckEntityExists) answers 204; every other APIM HEAD answers 200.
// Source: armapimanagement v1.1.1 accepted status codes.
var azureAPIManagementCheckEntityExistsRoutes = [][2]string{
	{"groups", "users"},
	{"notifications", "recipientEmails"},
	{"notifications", "recipientUsers"},
	{"products", "apis"},
	{"products", "groups"},
}

func azureAPIManagementIsCheckEntityExists(segments []string) bool {
	for idx := 0; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "Microsoft.ApiManagement") || !strings.EqualFold(segments[idx+1], "service") {
			continue
		}
		tail := segments[idx+2:]
		if len(tail) != 5 { // {serviceName}/{parent}/{id}/{child}/{id}
			return false
		}
		for _, route := range azureAPIManagementCheckEntityExistsRoutes {
			if strings.EqualFold(tail[1], route[0]) && strings.EqualFold(tail[3], route[1]) {
				return true
			}
		}
		return false
	}
	return false
}

func azureIsAPIManagementResourceManagerPath(segments []string) bool {
	// Provider-level endpoints:
	// /providers/Microsoft.ApiManagement/operations
	if len(segments) >= 2 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ApiManagement") {
		return true
	}

	// Subscription/resource-group scoped ARM paths:
	// /subscriptions/{id}/.../providers/Microsoft.ApiManagement/...
	if len(segments) < 2 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.ApiManagement") {
			return true
		}
	}
	return false
}

func azureAPIManagementResourceManagerAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return true
	default:
		return false
	}
}
