package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPAPIHubRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v1/projects/") || !strings.Contains(path, "/locations/") {
		return false
	}

	// Route recognition for API Hub v1 resources.
	// Note: top-level /apis is intentionally excluded here to avoid
	// colliding with API Gateway's /apis route in the shared /gcp mux.
	_, deploymentsCollection := parseGCPAPIHubLocationCollectionPath(path, "deployments")
	isAPIHubPath := strings.Contains(path, ":searchResources") ||
		strings.Contains(path, "/attributes") ||
		strings.Contains(path, "/externalApis") ||
		deploymentsCollection ||
		(strings.Contains(path, "/apis/") && (strings.Contains(path, "/versions") ||
			strings.Contains(path, "/specs") ||
			strings.Contains(path, "/operations") ||
			strings.Contains(path, "/definitions")))
	if !isAPIHubPath {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPAPIHubListAttributes(w, r, path) {
			return true
		}
		if handleGCPAPIHubListDeployments(w, r, path) {
			return true
		}
		if handleGCPAPIHubListExternalAPIs(w, r, path) {
			return true
		}
		if handleGCPAPIHubListVersions(w, r, path) {
			return true
		}
		if handleGCPAPIHubGetDefinition(w, path) {
			return true
		}
		if handleGCPAPIHubGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPAPIHubSearchResources(w, r, path) {
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

func handleGCPAPIHubListAttributes(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPAPIHubLocationCollectionPath(path, "attributes")
	if !ok {
		return false
	}
	return respondGCPAPIHubSimpleList(w, r, path, "attributes", []map[string]any{
		{
			"name":        fmt.Sprintf("%s/attributes/owner", parent),
			"displayName": "Owner",
		},
	})
}

func handleGCPAPIHubListDeployments(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPAPIHubLocationCollectionPath(path, "deployments")
	if !ok {
		return false
	}
	return respondGCPAPIHubSimpleList(w, r, path, "deployments", []map[string]any{
		{
			"name":        fmt.Sprintf("%s/deployments/team-api-us", parent),
			"displayName": "Team API US Deployment",
		},
	})
}

func handleGCPAPIHubListExternalAPIs(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPAPIHubLocationCollectionPath(path, "externalApis")
	if !ok {
		return false
	}
	return respondGCPAPIHubSimpleList(w, r, path, "externalApis", []map[string]any{
		{
			"name":        fmt.Sprintf("%s/externalApis/orders", parent),
			"displayName": "Orders API",
		},
	})
}

func handleGCPAPIHubListVersions(w http.ResponseWriter, r *http.Request, path string) bool {
	apiName, ok := parseGCPAPIHubAPIVersionsCollectionPath(path)
	if !ok {
		return false
	}
	return respondGCPAPIHubSimpleList(w, r, path, "versions", []map[string]any{
		{
			"name":        fmt.Sprintf("%s/versions/v1", apiName),
			"displayName": "v1",
		},
	})
}

func handleGCPAPIHubGetDefinition(w http.ResponseWriter, path string) bool {
	versionName, definitionID, ok := parseGCPAPIHubDefinitionPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":        fmt.Sprintf("%s/definitions/%s", versionName, definitionID),
		"displayName": "OpenAPI",
		"description": "OpenAPI definition for team API",
	})
	return true
}

func handleGCPAPIHubGetOperation(w http.ResponseWriter, path string) bool {
	versionName, operationID, ok := parseGCPAPIHubOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":        fmt.Sprintf("%s/operations/%s", versionName, operationID),
		"displayName": "GetOrders",
		"description": "Retrieves orders",
	})
	return true
}

func handleGCPAPIHubSearchResources(w http.ResponseWriter, r *http.Request, path string) bool {
	location, ok := parseGCPAPIHubSearchPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPAPIHubJSONBody(w, r, path)
	if !valid {
		return true
	}
	query, _ := body["query"].(string)
	if strings.TrimSpace(query) == "" {
		respondGCPAPIHubInvalidArgument(w, path, "query is required")
		return true
	}

	pageSize, err := parseOptionalNonNegativeInt(getStringFromAny(body["pageSize"]))
	if err != nil {
		respondGCPAPIHubInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}

	results := []map[string]any{
		{
			"name":         fmt.Sprintf("%s/apis/team-api", location),
			"resourceType": "API",
		},
	}
	end := len(results)
	if pageSize > 0 && pageSize < end {
		end = pageSize
	}
	nextPageToken := ""
	if end < len(results) {
		nextPageToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"results":       results[:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func parseGCPAPIHubLocationCollectionPath(path, collection string) (parent string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 {
		return "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != collection {
		return "", false
	}
	if strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(parts[5]) == "" {
		return "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[3], parts[5]), true
}

func parseGCPAPIHubAPIVersionsCollectionPath(path string) (apiName string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 {
		return "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "apis" || parts[8] != "versions" {
		return "", false
	}
	if strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(parts[5]) == "" || strings.TrimSpace(parts[7]) == "" {
		return "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s/apis/%s", parts[3], parts[5], parts[7]), true
}

func parseGCPAPIHubDefinitionPath(path string) (versionName, definitionID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 12 {
		return "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "apis" || parts[8] != "versions" || parts[10] != "definitions" {
		return "", "", false
	}
	if strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(parts[5]) == "" || strings.TrimSpace(parts[7]) == "" || strings.TrimSpace(parts[9]) == "" || strings.TrimSpace(parts[11]) == "" {
		return "", "", false
	}
	versionName = fmt.Sprintf("projects/%s/locations/%s/apis/%s/versions/%s", parts[3], parts[5], parts[7], parts[9])
	return versionName, parts[11], true
}

func parseGCPAPIHubOperationPath(path string) (versionName, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 12 {
		return "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "apis" || parts[8] != "versions" || parts[10] != "operations" {
		return "", "", false
	}
	if strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(parts[5]) == "" || strings.TrimSpace(parts[7]) == "" || strings.TrimSpace(parts[9]) == "" || strings.TrimSpace(parts[11]) == "" {
		return "", "", false
	}
	versionName = fmt.Sprintf("projects/%s/locations/%s/apis/%s/versions/%s", parts[3], parts[5], parts[7], parts[9])
	return versionName, parts[11], true
}

func parseGCPAPIHubSearchPath(path string) (location string, ok bool) {
	normalized := strings.ReplaceAll(strings.ReplaceAll(path, "%3A", ":"), "%3a", ":")
	parts := strings.Split(strings.Trim(normalized, "/"), "/")
	if len(parts) != 6 {
		return "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", false
	}
	locAction := parts[5]
	locationID, action, found := strings.Cut(locAction, ":")
	if !found || action != "searchResources" {
		return "", false
	}
	if strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(locationID) == "" {
		return "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[3], locationID), true
}

func respondGCPAPIHubSimpleList(w http.ResponseWriter, r *http.Request, path, key string, items []map[string]any) bool {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPAPIHubInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondGCPAPIHubInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return true
		}
	}
	if start > len(items) {
		respondGCPAPIHubInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPAPIHubJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPAPIHubInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func getStringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.Itoa(int(v))
	default:
		return ""
	}
}

func respondGCPAPIHubInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
