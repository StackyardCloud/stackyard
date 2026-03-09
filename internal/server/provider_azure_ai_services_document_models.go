package server

import (
	"net/http"
	"strings"
)

const azureAIServicesDocumentModelsPrefix = "/azure/aiservices/documentintelligence/"

func (s *Server) handleAzureAIServicesDocumentModelsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureAIServicesDocumentModelsPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureAIServicesDocumentModelsPrefix)
	normalized := strings.TrimSuffix(strings.TrimPrefix(relative, "/"), "/")
	if normalized == "" {
		return false
	}

	switch normalized {
	case "documentModels:authorizeCopy":
		if r.Method == http.MethodPost {
			respondAzureImplemented(w, path)
			return true
		}
	case "documentModels:build":
		if r.Method == http.MethodPost {
			respondAzureImplemented(w, path)
			return true
		}
	case "documentModels:compose":
		if r.Method == http.MethodPost {
			respondAzureImplemented(w, path)
			return true
		}
	case "documentModels":
		if r.Method == http.MethodGet {
			respondAzureImplemented(w, path)
			return true
		}
	}

	if !strings.HasPrefix(normalized, "documentModels/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(normalized, "documentModels/"))
	if len(segments) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	if modelID, operation, ok := parseAzureDocumentModelToken(segments[0]); ok {
		switch {
		case modelID != "" && operation == "" && len(segments) == 1 && (r.Method == http.MethodGet || r.Method == http.MethodDelete):
			respondAzureImplemented(w, path)
			return true
		case modelID != "" && operation == "copyTo" && len(segments) == 1 && r.Method == http.MethodPost:
			respondAzureImplemented(w, path)
			return true
		case modelID != "" && operation == "analyze" && len(segments) == 1 && r.Method == http.MethodPost:
			respondAzureImplemented(w, path)
			return true
		case modelID != "" && operation == "analyzeBatch" && len(segments) == 1 && r.Method == http.MethodPost:
			respondAzureImplemented(w, path)
			return true
		}
	}

	if len(segments) == 2 && segments[1] == "analyzeBatchResults" && r.Method == http.MethodGet {
		respondAzureImplemented(w, path)
		return true
	}

	if len(segments) == 3 && segments[1] == "analyzeBatchResults" && (r.Method == http.MethodGet || r.Method == http.MethodDelete) {
		respondAzureImplemented(w, path)
		return true
	}

	if len(segments) == 3 && segments[1] == "analyzeResults" && (r.Method == http.MethodGet || r.Method == http.MethodDelete) {
		respondAzureImplemented(w, path)
		return true
	}

	if len(segments) == 4 && segments[1] == "analyzeResults" && segments[3] == "pdf" && r.Method == http.MethodGet {
		respondAzureImplemented(w, path)
		return true
	}

	if len(segments) == 5 && segments[1] == "analyzeResults" && segments[3] == "figures" && r.Method == http.MethodGet {
		respondAzureImplemented(w, path)
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func parseAzureDocumentModelToken(token string) (modelID, operation string, ok bool) {
	id, op, hasOperation := strings.Cut(token, ":")
	modelID = strings.TrimSpace(id)
	operation = strings.TrimSpace(op)
	if !hasOperation {
		return modelID, "", true
	}
	return modelID, operation, true
}
