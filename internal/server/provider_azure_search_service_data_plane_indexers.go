package server

import (
	"net/http"
	"strings"
)

const azureSearchServiceIndexersBase = "/azure/indexers"

func (s *Server) handleAzureSearchServiceDataPlaneIndexersRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSearchServiceIndexersBase) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if len(path) == len(azureSearchServiceIndexersBase) {
		respondAzureImplemented(w, path)
		return true
	}

	switch path[len(azureSearchServiceIndexersBase)] {
	case '(', '/':
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}
