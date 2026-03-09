package server

import "net/http"

const azureLanguageAnalyzeTextPath = "/azure/language/:analyze-text"

func (s *Server) handleAzureAIServicesLanguageAnalyzeTextRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if path != azureLanguageAnalyzeTextPath {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	respondAzureImplemented(w, path)
	return true
}
