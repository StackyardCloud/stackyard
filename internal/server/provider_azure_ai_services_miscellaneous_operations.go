package server

import (
	"net/http"
	"strings"
)

const azureAIServicesDocumentIntelligenceMiscPrefix = "/azure/aiservices/documentintelligence/"

func (s *Server) handleAzureAIServicesMiscellaneousOperationsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureAIServicesDocumentIntelligenceMiscPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureAIServicesDocumentIntelligenceMiscPrefix)
	normalized := strings.TrimSuffix(strings.TrimPrefix(relative, "/"), "/")
	if normalized == "" {
		return false
	}

	if normalized == "info" {
		respondAzureImplemented(w, path)
		return true
	}

	if normalized == "operations" {
		respondAzureImplemented(w, path)
		return true
	}

	if operationID, ok := parseAzureAIServicesOperationIDPath(normalized); ok && operationID != "" {
		respondAzureImplemented(w, path)
		return true
	}

	return false
}

func parseAzureAIServicesOperationIDPath(normalizedPath string) (operationID string, ok bool) {
	if !strings.HasPrefix(normalizedPath, "operations/") {
		return "", false
	}
	tail := strings.TrimPrefix(normalizedPath, "operations/")
	if strings.Contains(tail, "/") {
		return "", false
	}
	return strings.TrimSpace(tail), true
}
