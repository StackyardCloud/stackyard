package server

import (
	"net/http"
	"strings"
)

const azureLanguageAnalyzeTextAuthoringPrefix = "/azure/language/authoring/analyze-text"

func (s *Server) handleAzureAIServicesLanguageAnalyzeTextAuthoringRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if path != azureLanguageAnalyzeTextAuthoringPrefix && !strings.HasPrefix(path, azureLanguageAnalyzeTextAuthoringPrefix+"/") {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	respondAzureImplemented(w, path)
	return true
}
