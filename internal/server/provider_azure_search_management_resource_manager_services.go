package server

import (
	"net/http"
	"strings"
)

const azureSearchManagementServicesSubscriptionsBase = "/azure/subscriptions/"

func (s *Server) handleAzureSearchManagementResourceManagerServicesRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSearchManagementServicesSubscriptionsBase) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) < 5 {
		return false
	}

	// Subscription-scoped operations
	if strings.EqualFold(segments[0], "subscriptions") &&
		segments[1] != "" &&
		strings.EqualFold(segments[2], "providers") &&
		strings.EqualFold(segments[3], "Microsoft.Search") {
		if len(segments) == 5 && segments[4] == "checkNameAvailability" && r.Method == http.MethodPost {
			respondAzureImplemented(w, path)
			return true
		}
		if len(segments) == 5 && segments[4] == "searchServices" && r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	}

	// Resource-group scoped service operations
	if len(segments) < 7 {
		return false
	}
	if !strings.EqualFold(segments[0], "subscriptions") ||
		segments[1] == "" ||
		!strings.EqualFold(segments[2], "resourceGroups") ||
		segments[3] == "" ||
		!strings.EqualFold(segments[4], "providers") ||
		!strings.EqualFold(segments[5], "Microsoft.Search") ||
		segments[6] != "searchServices" {
		return false
	}

	if len(segments) == 7 && r.Method == http.MethodGet {
		respondAzureImplemented(w, path)
		return true
	}
	if len(segments) == 8 && segments[7] != "" {
		switch r.Method {
		case http.MethodPut, http.MethodDelete, http.MethodGet, http.MethodPatch:
			respondAzureImplemented(w, path)
			return true
		}
	}
	if len(segments) == 9 && segments[7] != "" && segments[8] == "upgrade" && r.Method == http.MethodPost {
		respondAzureImplemented(w, path)
		return true
	}

	return false
}
