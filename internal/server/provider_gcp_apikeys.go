package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPAPIKeysRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)

	// Route recognition for API Keys v2 resources:
	// - /v2/projects/*/locations/*/keys/*
	// - /v2/keys:lookupKey
	isAPIKeysPath := path == "/gcp/v2/keys:lookupKey" ||
		(strings.HasPrefix(path, "/gcp/v2/projects/") &&
			strings.Contains(path, "/locations/") &&
			strings.Contains(path, "/keys"))
	if !isAPIKeysPath {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPAPIKeysListKeys(w, r, path) {
			return true
		}
		if handleGCPAPIKeysGetKey(w, path) {
			return true
		}
		if handleGCPAPIKeysGetKeyString(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPAPIKeysLookupKey(w, r, path) {
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

func handleGCPAPIKeysListKeys(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPAPIKeysCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPAPIKeysInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondGCPAPIKeysInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return true
		}
	}

	items := []map[string]any{
		gcpAPIKeysKey(parent, "team-key"),
	}
	if start > len(items) {
		respondGCPAPIKeysInvalidArgument(w, path, "pageToken is out of range")
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
		"keys":          items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func handleGCPAPIKeysGetKey(w http.ResponseWriter, path string) bool {
	parent, keyID, keyStringEndpoint, ok := parseGCPAPIKeysResourcePath(path)
	if !ok || keyStringEndpoint {
		return false
	}
	respondJSON(w, http.StatusOK, gcpAPIKeysKey(parent, keyID))
	return true
}

func handleGCPAPIKeysGetKeyString(w http.ResponseWriter, path string) bool {
	parent, keyID, keyStringEndpoint, ok := parseGCPAPIKeysResourcePath(path)
	if !ok || !keyStringEndpoint {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"keyString": "stackyard-demo-key",
		"name":      fmt.Sprintf("%s/keys/%s", parent, keyID),
	})
	return true
}

func handleGCPAPIKeysLookupKey(w http.ResponseWriter, r *http.Request, path string) bool {
	normalized := strings.ReplaceAll(strings.ReplaceAll(path, "%3A", ":"), "%3a", ":")
	if normalized != "/gcp/v2/keys:lookupKey" {
		return false
	}

	body, valid := decodeGCPAPIKeysJSONBody(w, r, path)
	if !valid {
		return true
	}
	keyString, _ := body["keyString"].(string)
	if strings.TrimSpace(keyString) == "" {
		respondGCPAPIKeysInvalidArgument(w, path, "keyString is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": "projects/stackyard/locations/global/keys/team-key",
	})
	return true
}

func parseGCPAPIKeysCollectionPath(path string) (parent string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 {
		return "", false
	}
	if parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "keys" {
		return "", false
	}
	if strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(parts[5]) == "" {
		return "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[3], parts[5]), true
}

func parseGCPAPIKeysResourcePath(path string) (parent, keyID string, keyStringEndpoint bool, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 && len(parts) != 9 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "keys" {
		return "", "", false, false
	}
	if strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(parts[5]) == "" || strings.TrimSpace(parts[7]) == "" {
		return "", "", false, false
	}
	parent = fmt.Sprintf("projects/%s/locations/%s", parts[3], parts[5])
	keyID = parts[7]
	if len(parts) == 9 {
		if parts[8] != "keyString" {
			return "", "", false, false
		}
		return parent, keyID, true, true
	}
	return parent, keyID, false, true
}

func gcpAPIKeysKey(parent, keyID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("%s/keys/%s", parent, keyID),
		"displayName": "Team Key",
		"keyString":   "stackyard-demo-key",
	}
}

func decodeGCPAPIKeysJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPAPIKeysInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func respondGCPAPIKeysInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
