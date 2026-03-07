package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPDiscoveryEngineRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPDiscoveryEnginePath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.cloud.discoveryengine.v1.") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPDiscoveryEngineListEngines(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineGetEngine(w, path) {
			return true
		}
		if handleGCPDiscoveryEngineListDataStores(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineGetDataStore(w, path) {
			return true
		}
		if handleGCPDiscoveryEngineListConversations(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineGetConversation(w, path) {
			return true
		}
		if handleGCPDiscoveryEngineListSessions(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineGetSession(w, path) {
			return true
		}
		if handleGCPDiscoveryEngineListOperations(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPDiscoveryEngineCreateEngine(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineCreateDataStore(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineSearch(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineSearchLite(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineCompleteQuery(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineAnswer(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineCreateConversation(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineConverse(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineCreateSession(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineCancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPDiscoveryEngineUpdateEngine(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineUpdateDataStore(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineUpdateConversation(w, r, path) {
			return true
		}
		if handleGCPDiscoveryEngineUpdateSession(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPDiscoveryEngineDeleteEngine(w, path) {
			return true
		}
		if handleGCPDiscoveryEngineDeleteDataStore(w, path) {
			return true
		}
		if handleGCPDiscoveryEngineDeleteSession(w, path) {
			return true
		}
		if handleGCPDiscoveryEngineDeleteConversation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPDiscoveryEnginePath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.discoveryengine.v1.") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/") {
		return false
	}
	if strings.HasPrefix(path, "/gcp/v1/providers/") || strings.HasPrefix(path, "/gcp/v1/places") {
		return false
	}
	return strings.Contains(path, "/collections/") &&
		(strings.Contains(path, "/engines") ||
			strings.Contains(path, "/dataStores") ||
			strings.Contains(path, "/servingConfigs") ||
			strings.Contains(path, "/conversations") ||
			strings.Contains(path, "/sessions") ||
			strings.Contains(path, "/operations") ||
			strings.Contains(path, ":search") ||
			strings.Contains(path, ":searchLite") ||
			strings.Contains(path, ":completeQuery") ||
			strings.Contains(path, ":answer") ||
			strings.Contains(path, ":converse") ||
			strings.Contains(path, ":cancel"))
}

func handleGCPDiscoveryEngineListEngines(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 1 || tail[0] != "engines" {
		return false
	}
	pageSize, start, valid := parseGCPDiscoveryEnginePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDiscoveryEngineEngine(project, location, collection, "orders-engine")}
	return respondGCPDiscoveryEngineList(w, "engines", items, pageSize, start, path)
}

func handleGCPDiscoveryEngineGetEngine(w http.ResponseWriter, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 2 || tail[0] != "engines" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineEngine(project, location, collection, tail[1]))
	return true
}

func handleGCPDiscoveryEngineCreateEngine(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 1 || tail[0] != "engines" {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	engine := gcpDiscoveryEngineBodyMap(body, "engine")
	if len(engine) == 0 {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "engine is required")
		return true
	}
	engineID := strings.TrimSpace(r.URL.Query().Get("engineId"))
	if engineID == "" {
		engineID = "orders-engine"
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineOperation(project, location, collection, "create-engine-"+engineID))
	return true
}

func handleGCPDiscoveryEngineUpdateEngine(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 2 || tail[0] != "engines" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	engine := gcpDiscoveryEngineBodyMap(body, "engine")
	if len(engine) == 0 {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "engine is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineEngine(project, location, collection, tail[1]))
	return true
}

func handleGCPDiscoveryEngineDeleteEngine(w http.ResponseWriter, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 2 || tail[0] != "engines" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineOperation(project, location, collection, "delete-engine-"+tail[1]))
	return true
}

func handleGCPDiscoveryEngineListDataStores(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 1 || tail[0] != "dataStores" {
		return false
	}
	pageSize, start, valid := parseGCPDiscoveryEnginePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDiscoveryEngineDataStore(project, location, collection, "orders-store")}
	return respondGCPDiscoveryEngineList(w, "dataStores", items, pageSize, start, path)
}

func handleGCPDiscoveryEngineGetDataStore(w http.ResponseWriter, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 2 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineDataStore(project, location, collection, tail[1]))
	return true
}

func handleGCPDiscoveryEngineCreateDataStore(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 1 || tail[0] != "dataStores" {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	dataStore := gcpDiscoveryEngineBodyMap(body, "dataStore")
	if len(dataStore) == 0 {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "dataStore is required")
		return true
	}
	dataStoreID := strings.TrimSpace(r.URL.Query().Get("dataStoreId"))
	if dataStoreID == "" {
		dataStoreID = "orders-store"
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineOperation(project, location, collection, "create-datastore-"+dataStoreID))
	return true
}

func handleGCPDiscoveryEngineUpdateDataStore(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 2 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	dataStore := gcpDiscoveryEngineBodyMap(body, "dataStore")
	if len(dataStore) == 0 {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "dataStore is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineDataStore(project, location, collection, tail[1]))
	return true
}

func handleGCPDiscoveryEngineDeleteDataStore(w http.ResponseWriter, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 2 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineOperation(project, location, collection, "delete-datastore-"+tail[1]))
	return true
}

func handleGCPDiscoveryEngineSearch(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 4 || tail[0] != "engines" || strings.TrimSpace(tail[1]) == "" || tail[2] != "servingConfigs" || tail[3] == "" {
		return false
	}
	configID, action, hasAction := strings.Cut(normalizeGCPDiscoveryEngineActionSegment(tail[3]), ":")
	if !hasAction || strings.TrimSpace(configID) == "" || action != "search" {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(stringFromMap(body, "query")) == "" {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "query is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"results": []any{
			map[string]any{"id": "doc-1", "document": map[string]any{"name": fmt.Sprintf("projects/%s/locations/%s/collections/%s/dataStores/orders-store/branches/default_branch/documents/doc-1", project, location, collection)}},
		},
		"nextPageToken": "",
	})
	return true
}

func handleGCPDiscoveryEngineSearchLite(w http.ResponseWriter, r *http.Request, path string) bool {
	if !strings.Contains(path, ":searchLite") {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(stringFromMap(body, "query")) == "" {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "query is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"results":       []any{map[string]any{"id": "lite-1"}},
		"nextPageToken": "",
	})
	return true
}

func handleGCPDiscoveryEngineCompleteQuery(w http.ResponseWriter, r *http.Request, path string) bool {
	if !strings.Contains(path, ":completeQuery") {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(stringFromMap(body, "query")) == "" {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "query is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"querySuggestions": []any{
			map[string]any{"suggestion": "order status"},
		},
	})
	return true
}

func handleGCPDiscoveryEngineAnswer(w http.ResponseWriter, r *http.Request, path string) bool {
	if !strings.Contains(path, ":answer") {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	query, _ := body["query"].(map[string]any)
	if len(query) == 0 {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "query is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"answer": map[string]any{
			"state":    "SUCCEEDED",
			"answer":   "Order o-1 is delivered.",
			"grounded": true,
		},
	})
	return true
}

func handleGCPDiscoveryEngineListConversations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 3 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || tail[2] != "conversations" {
		return false
	}
	pageSize, start, valid := parseGCPDiscoveryEnginePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDiscoveryEngineConversation(project, location, collection, tail[1], "conv-1")}
	return respondGCPDiscoveryEngineList(w, "conversations", items, pageSize, start, path)
}

func handleGCPDiscoveryEngineCreateConversation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 3 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || tail[2] != "conversations" {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	conversation := gcpDiscoveryEngineBodyMap(body, "conversation")
	if len(conversation) == 0 {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "conversation is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineConversation(project, location, collection, tail[1], "conv-1"))
	return true
}

func handleGCPDiscoveryEngineGetConversation(w http.ResponseWriter, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 4 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || tail[2] != "conversations" || strings.TrimSpace(tail[3]) == "" || strings.Contains(tail[3], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineConversation(project, location, collection, tail[1], tail[3]))
	return true
}

func handleGCPDiscoveryEngineUpdateConversation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 4 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || tail[2] != "conversations" || strings.TrimSpace(tail[3]) == "" || strings.Contains(tail[3], ":") {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	conversation := gcpDiscoveryEngineBodyMap(body, "conversation")
	if len(conversation) == 0 {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "conversation is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineConversation(project, location, collection, tail[1], tail[3]))
	return true
}

func handleGCPDiscoveryEngineConverse(w http.ResponseWriter, r *http.Request, path string) bool {
	if !strings.Contains(path, ":converse") {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	if _, ok := body["query"]; !ok {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "query is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"reply": map[string]any{
			"reply": "Here are your recent orders.",
		},
	})
	return true
}

func handleGCPDiscoveryEngineDeleteConversation(w http.ResponseWriter, path string) bool {
	_, _, _, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 4 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || tail[2] != "conversations" || strings.TrimSpace(tail[3]) == "" || strings.Contains(tail[3], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDiscoveryEngineListSessions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 3 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || tail[2] != "sessions" {
		return false
	}
	pageSize, start, valid := parseGCPDiscoveryEnginePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDiscoveryEngineSession(project, location, collection, tail[1], "session-1")}
	return respondGCPDiscoveryEngineList(w, "sessions", items, pageSize, start, path)
}

func handleGCPDiscoveryEngineCreateSession(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 3 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || tail[2] != "sessions" {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	session := gcpDiscoveryEngineBodyMap(body, "session")
	if len(session) == 0 {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "session is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineSession(project, location, collection, tail[1], "session-1"))
	return true
}

func handleGCPDiscoveryEngineGetSession(w http.ResponseWriter, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 4 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || tail[2] != "sessions" || strings.TrimSpace(tail[3]) == "" || strings.Contains(tail[3], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineSession(project, location, collection, tail[1], tail[3]))
	return true
}

func handleGCPDiscoveryEngineUpdateSession(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 4 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || tail[2] != "sessions" || strings.TrimSpace(tail[3]) == "" || strings.Contains(tail[3], ":") {
		return false
	}
	body, valid := decodeGCPDiscoveryEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	session := gcpDiscoveryEngineBodyMap(body, "session")
	if len(session) == 0 {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "session is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineSession(project, location, collection, tail[1], tail[3]))
	return true
}

func handleGCPDiscoveryEngineDeleteSession(w http.ResponseWriter, path string) bool {
	_, _, _, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 4 || tail[0] != "dataStores" || strings.TrimSpace(tail[1]) == "" || tail[2] != "sessions" || strings.TrimSpace(tail[3]) == "" || strings.Contains(tail[3], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDiscoveryEngineListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 1 || tail[0] != "operations" {
		return false
	}
	pageSize, start, valid := parseGCPDiscoveryEnginePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDiscoveryEngineOperation(project, location, collection, "op-1")}
	return respondGCPDiscoveryEngineList(w, "operations", items, pageSize, start, path)
}

func handleGCPDiscoveryEngineGetOperation(w http.ResponseWriter, path string) bool {
	project, location, collection, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDiscoveryEngineOperation(project, location, collection, tail[1]))
	return true
}

func handleGCPDiscoveryEngineCancelOperation(w http.ResponseWriter, path string) bool {
	_, _, _, tail, ok := parseGCPDiscoveryEngineCollectionTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	operationID, action, hasAction := strings.Cut(normalizeGCPDiscoveryEngineActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(operationID) == "" || action != "cancel" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPDiscoveryEngineCollectionTail(path string) (project, location, collection string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 9 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "collections" {
		return "", "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	collection = strings.TrimSpace(parts[7])
	if project == "" || location == "" || collection == "" {
		return "", "", "", nil, false
	}
	tail = parts[8:]
	return project, location, collection, tail, len(tail) > 0
}

func parseGCPDiscoveryEnginePagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPDiscoveryEngineList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPDiscoveryEngineJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPDiscoveryEngineInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpDiscoveryEngineBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func normalizeGCPDiscoveryEngineActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func gcpDiscoveryEngineEngine(project, location, collection, engineID string) map[string]any {
	return map[string]any{
		"name":         fmt.Sprintf("projects/%s/locations/%s/collections/%s/engines/%s", project, location, collection, engineID),
		"displayName":  engineID,
		"solutionType": "SOLUTION_TYPE_SEARCH",
	}
}

func gcpDiscoveryEngineDataStore(project, location, collection, dataStoreID string) map[string]any {
	return map[string]any{
		"name":             fmt.Sprintf("projects/%s/locations/%s/collections/%s/dataStores/%s", project, location, collection, dataStoreID),
		"displayName":      dataStoreID,
		"industryVertical": "GENERIC",
	}
}

func gcpDiscoveryEngineConversation(project, location, collection, dataStoreID, conversationID string) map[string]any {
	return map[string]any{
		"name":         fmt.Sprintf("projects/%s/locations/%s/collections/%s/dataStores/%s/conversations/%s", project, location, collection, dataStoreID, conversationID),
		"userPseudoId": "user-1",
	}
}

func gcpDiscoveryEngineSession(project, location, collection, dataStoreID, sessionID string) map[string]any {
	return map[string]any{
		"name":         fmt.Sprintf("projects/%s/locations/%s/collections/%s/dataStores/%s/sessions/%s", project, location, collection, dataStoreID, sessionID),
		"displayName":  sessionID,
		"userPseudoId": "user-1",
	}
}

func gcpDiscoveryEngineOperation(project, location, collection, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/collections/%s/operations/%s", project, location, collection, operationID),
		"done": true,
	}
}

func respondGCPDiscoveryEngineInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
