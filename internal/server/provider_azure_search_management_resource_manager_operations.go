package server

import (
	"net/http"
	"strings"
)

const azureSearchManagementOperationsBase = "/azure/providers/"

func (s *Server) handleAzureSearchManagementResourceManagerOperationsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSearchManagementOperationsBase) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) != 3 {
		return false
	}
	if !strings.EqualFold(segments[0], "providers") ||
		!strings.EqualFold(segments[1], "Microsoft.Search") ||
		segments[2] != "operations" {
		return false
	}
	if r.Method == http.MethodGet {
		respondAzureImplemented(w, path)
		return true
	}
	return false
}
