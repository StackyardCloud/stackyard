package server

import (
	"net/http"
	"strings"
)

const azureLanguageQuestionAnsweringAuthoringPrefix = "/azure/language/authoring/query-knowledgebases"

func (s *Server) handleAzureAIServicesLanguageQuestionAnsweringAuthoringRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if path != azureLanguageQuestionAnsweringAuthoringPrefix && !strings.HasPrefix(path, azureLanguageQuestionAnsweringAuthoringPrefix+"/") {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	respondAzureImplemented(w, path)
	return true
}
