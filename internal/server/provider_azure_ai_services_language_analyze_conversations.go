package server

import (
	"net/http"
	"strings"
)

const (
	azureLanguageAnalyzeConversationsPath       = "/azure/language/:analyze-conversations"
	azureLanguageAnalyzeConversationsJobsPath   = "/azure/language/analyze-conversations/jobs"
	azureLanguageAnalyzeConversationsJobsPrefix = "/azure/language/analyze-conversations/jobs/"
)

func (s *Server) handleAzureAIServicesLanguageAnalyzeConversationsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	switch {
	case path == azureLanguageAnalyzeConversationsPath,
		path == azureLanguageAnalyzeConversationsJobsPath,
		strings.HasPrefix(path, azureLanguageAnalyzeConversationsJobsPrefix):
		if hasAzureInvalidAPIVersion(r) {
			respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
			return true
		}
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}
