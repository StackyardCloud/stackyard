package server

import (
	"net/http"
	"strings"
)

const azureSearchServiceIndexesBase = "/azure/indexes"

func (s *Server) handleAzureSearchServiceDataPlaneIndexesRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSearchServiceIndexesBase) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if len(path) == len(azureSearchServiceIndexesBase) {
		respondAzureImplemented(w, path)
		return true
	}

	switch path[len(azureSearchServiceIndexesBase)] {
	case '(', '/':
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}
