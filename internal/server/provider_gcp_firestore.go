package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) handleGCPFirestoreRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPFirestorePath(path) {
		return false
	}
	if strings.HasPrefix(path, "/gcp/google.firestore.v1.Firestore/") {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPFirestoreListDocuments(w, r, path) {
			return true
		}
		if handleGCPFirestoreGetDocument(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPFirestoreCreateDocument(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPFirestorePath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.firestore.v1.Firestore/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/databases/") {
		return false
	}

	return strings.Contains(path, "/documents") ||
		strings.Contains(path, ":batchGet") ||
		strings.Contains(path, ":beginTransaction") ||
		strings.Contains(path, ":commit") ||
		strings.Contains(path, ":rollback") ||
		strings.Contains(path, ":runQuery") ||
		strings.Contains(path, ":runAggregationQuery") ||
		strings.Contains(path, ":partitionQuery") ||
		strings.Contains(path, ":listCollectionIds") ||
		strings.Contains(path, ":batchWrite") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":cancel")
}

func handleGCPFirestoreListDocuments(w http.ResponseWriter, r *http.Request, path string) bool {
	project, database, collection, ok := parseGCPFirestoreCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "pageToken must be a non-negative integer offset",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
	}

	docs := []map[string]any{
		gcpFirestoreDocument(project, database, collection, "user-1", "stackyard-user-1"),
		gcpFirestoreDocument(project, database, collection, "user-2", "stackyard-user-2"),
	}
	if start > len(docs) {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageToken is out of range",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	end := len(docs)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(docs) {
		nextPageToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"documents":     docs[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func handleGCPFirestoreGetDocument(w http.ResponseWriter, path string) bool {
	project, database, collection, documentID, ok := parseGCPFirestoreDocumentPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpFirestoreDocument(project, database, collection, documentID, "stackyard-user"))
	return true
}

func handleGCPFirestoreCreateDocument(w http.ResponseWriter, r *http.Request, path string) bool {
	project, database, collection, ok := parseGCPFirestoreCollectionPath(path)
	if !ok {
		return false
	}

	var req struct {
		Fields map[string]any `json:"fields"`
	}
	if r.Body != nil {
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		if err := decoder.Decode(&req); err != nil && err.Error() != "EOF" {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "request body must be valid JSON",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
	}

	documentID := strings.TrimSpace(r.URL.Query().Get("documentId"))
	if documentID == "" {
		documentID = "user-1"
	}
	displayName := "stackyard-user"
	if raw, ok := req.Fields["displayName"].(map[string]any); ok {
		if val, ok := raw["stringValue"].(string); ok && strings.TrimSpace(val) != "" {
			displayName = strings.TrimSpace(val)
		}
	}

	respondJSON(w, http.StatusOK, gcpFirestoreDocument(project, database, collection, documentID, displayName))
	return true
}

func parseGCPFirestoreCollectionPath(path string) (project, database, collection string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	// /gcp/v1/projects/{project}/databases/{database}/documents/{collection}
	if len(parts) != 8 {
		return "", "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "databases" || parts[6] != "documents" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	database = strings.TrimSpace(parts[5])
	collection = strings.TrimSpace(parts[7])
	if project == "" || database == "" || collection == "" || strings.Contains(collection, ":") {
		return "", "", "", false
	}
	return project, database, collection, true
}

func parseGCPFirestoreDocumentPath(path string) (project, database, collection, documentID string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	// /gcp/v1/projects/{project}/databases/{database}/documents/{collection}/{document}
	if len(parts) != 9 {
		return "", "", "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "databases" || parts[6] != "documents" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	database = strings.TrimSpace(parts[5])
	collection = strings.TrimSpace(parts[7])
	documentID = strings.TrimSpace(parts[8])
	if project == "" || database == "" || collection == "" || documentID == "" || strings.Contains(collection, ":") || strings.Contains(documentID, ":") {
		return "", "", "", "", false
	}
	return project, database, collection, documentID, true
}

func gcpFirestoreDocument(project, database, collection, documentID, displayName string) map[string]any {
	name := "projects/" + project + "/databases/" + database + "/documents/" + collection + "/" + documentID
	ts := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	return map[string]any{
		"name": name,
		"fields": map[string]any{
			"displayName": map[string]any{
				"stringValue": displayName,
			},
		},
		"createTime": ts,
		"updateTime": ts,
	}
}
