package server

import "net/http"

const (
	azureLanguageQuestionAnsweringPath         = "/azure/language/:query-knowledgebases"
	azureLanguageQuestionAnsweringFromTextPath = "/azure/language/:query-text"
)

func (s *Server) handleAzureAIServicesLanguageQuestionAnsweringRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if path != azureLanguageQuestionAnsweringPath && path != azureLanguageQuestionAnsweringFromTextPath {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	respondAzureImplemented(w, path)
	return true
}
