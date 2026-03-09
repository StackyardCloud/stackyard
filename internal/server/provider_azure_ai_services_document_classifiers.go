package server

import (
	"net/http"
	"strings"
)

const azureAIServicesDocumentIntelligencePrefix = "/azure/aiservices/documentintelligence/"

func (s *Server) handleAzureAIServicesDocumentClassifiersRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureAIServicesDocumentIntelligencePrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureAIServicesDocumentIntelligencePrefix)
	normalized := strings.TrimSuffix(strings.TrimPrefix(relative, "/"), "/")
	if normalized == "" {
		respondAzureImplemented(w, path)
		return true
	}

	switch normalized {
	case "documentClassifiers:authorizeCopy":
		if r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{
				"classifierId":         "target-classifier",
				"accessToken":          "stackyard-copy-token",
				"expirationDateTime":   "2099-01-01T00:00:00Z",
				"targetResourceId":     "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.CognitiveServices/accounts/stackyard-target",
				"targetResourceRegion": "eastus",
				"provider":             providerAzure,
				"path":                 path,
			})
			return true
		}
	case "documentClassifiers:build":
		if r.Method == http.MethodPost {
			respondAzureAIServicesAccepted(w, path, "document-classifiers-build-op")
			return true
		}
	case "documentClassifiers":
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{
				"value": []map[string]any{
					{
						"classifierId":         "invoice-classifier",
						"description":          "stackyard local classifier",
						"createdDateTime":      "2026-01-01T00:00:00Z",
						"lastModifiedDateTime": "2026-01-01T00:00:00Z",
						"expirationDateTime":   "2099-01-01T00:00:00Z",
					},
				},
				"provider": providerAzure,
				"path":     path,
			})
			return true
		}
	}

	if classifierID, action, ok := parseAzureDocumentClassifierOperation(normalized); ok {
		switch {
		case classifierID != "" && action == "" && (r.Method == http.MethodGet || r.Method == http.MethodDelete):
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
				return true
			}
			respondJSON(w, http.StatusOK, map[string]any{
				"classifierId":         classifierID,
				"description":          "stackyard local classifier",
				"createdDateTime":      "2026-01-01T00:00:00Z",
				"lastModifiedDateTime": "2026-01-01T00:00:00Z",
				"expirationDateTime":   "2099-01-01T00:00:00Z",
				"provider":             providerAzure,
				"path":                 path,
			})
			return true
		case classifierID != "" && action == "analyze" && r.Method == http.MethodPost:
			respondAzureAIServicesAccepted(w, path, "document-classifiers-analyze-"+classifierID)
			return true
		case classifierID != "" && action == "copyTo" && r.Method == http.MethodPost:
			respondAzureAIServicesAccepted(w, path, "document-classifiers-copy-"+classifierID)
			return true
		}
		respondAzureImplemented(w, path)
		return true
	}

	if classifierID, resultID, ok := parseAzureDocumentClassifierAnalyzeResultPath(normalized); ok {
		if classifierID != "" && resultID != "" && r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{
				"resultId":     resultID,
				"classifierId": classifierID,
				"status":       "succeeded",
				"documents": []map[string]any{
					{
						"docType":    "invoice",
						"confidence": 0.99,
					},
				},
				"provider": providerAzure,
				"path":     path,
			})
			return true
		}
		respondAzureImplemented(w, path)
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func parseAzureDocumentClassifierOperation(normalizedPath string) (classifierID, action string, ok bool) {
	if !strings.HasPrefix(normalizedPath, "documentClassifiers/") {
		return "", "", false
	}
	tail := strings.TrimPrefix(normalizedPath, "documentClassifiers/")
	if strings.Contains(tail, "/") {
		return "", "", false
	}
	id, operation, hasOperation := strings.Cut(tail, ":")
	classifierID = strings.TrimSpace(id)
	action = strings.TrimSpace(operation)
	if !hasOperation {
		return classifierID, "", true
	}
	return classifierID, action, true
}

func parseAzureDocumentClassifierAnalyzeResultPath(normalizedPath string) (classifierID, resultID string, ok bool) {
	if !strings.HasPrefix(normalizedPath, "documentClassifiers/") {
		return "", "", false
	}
	segments := splitPathSegments(strings.TrimPrefix(normalizedPath, "documentClassifiers/"))
	if len(segments) != 3 || segments[1] != "analyzeResults" {
		return "", "", false
	}
	return strings.TrimSpace(segments[0]), strings.TrimSpace(segments[2]), true
}

func respondAzureAIServicesAccepted(w http.ResponseWriter, path, operationID string) {
	opID := strings.TrimSpace(operationID)
	if opID == "" {
		opID = "operation-1"
	}
	location := "/azure/aiservices/documentintelligence/operations/" + opID
	w.Header().Set("Operation-Location", location)
	w.Header().Set("Location", location)
	respondJSON(w, http.StatusAccepted, map[string]any{
		"operationId": opID,
		"status":      "running",
		"provider":    providerAzure,
		"path":        path,
	})
}
