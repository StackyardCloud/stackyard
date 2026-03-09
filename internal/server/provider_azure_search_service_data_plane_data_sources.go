package server

import (
	"net/http"
	"strings"
)

const azureSearchServiceDataSourcesBase = "/azure/datasources"

func (s *Server) handleAzureSearchServiceDataPlaneDataSourcesRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSearchServiceDataSourcesBase) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if len(path) == len(azureSearchServiceDataSourcesBase) {
		respondAzureImplemented(w, path)
		return true
	}

	switch path[len(azureSearchServiceDataSourcesBase)] {
	case '(', '/':
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}
