package server

import (
	"net/http"
	"strings"
)

const azureSearchServiceSkillsetsBase = "/azure/skillsets"

func (s *Server) handleAzureSearchServiceDataPlaneSkillsetsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSearchServiceSkillsetsBase) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if len(path) == len(azureSearchServiceSkillsetsBase) {
		respondAzureImplemented(w, path)
		return true
	}

	switch path[len(azureSearchServiceSkillsetsBase)] {
	case '(', '/':
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}
