package server

import (
	"net/http"
	"strings"
)

const azureSearchServiceSynonymMapsBase = "/azure/synonymmaps"

func (s *Server) handleAzureSearchServiceDataPlaneSynonymMapsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSearchServiceSynonymMapsBase) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if len(path) == len(azureSearchServiceSynonymMapsBase) {
		respondAzureImplemented(w, path)
		return true
	}

	switch path[len(azureSearchServiceSynonymMapsBase)] {
	case '(', '/':
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}
