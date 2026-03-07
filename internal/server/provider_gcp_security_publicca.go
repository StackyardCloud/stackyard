package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPSecurityPublicCARouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_security_publicca(w, r) {
		return true
	}

	path := normalizeGCPSecurityPublicCAPath(rawRequestPath(r))
	if isGCPSecurityPublicCALocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPSecurityPublicCAListLocations(w, r, path) {
			return true
		}
		if handleGCPSecurityPublicCAGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPSecurityPublicCAPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodPost:
		if handleGCPSecurityPublicCACreateExternalAccountKey(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSecurityPublicCAPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSecurityPublicCAHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "security_publicca", "security-publicca", "security-publicca-apiv1", "publicca", "public-ca", "public_ca", "publiccertificateauthority", "public-certificate-authority":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-security-publicca-apiv1") || strings.Contains(ua, "cloud.google.com/go/security/publicca")
}

func isGCPSecurityPublicCALocationRequest(r *http.Request, path string) bool {
	return hasGCPSecurityPublicCAHint(r) && isGCPProjectLocationDiscoveryPath(path)
}

func isGCPSecurityPublicCAPath(path string) bool {
	_, _, ok := parseGCPSecurityPublicCAExternalAccountKeysCollectionPath(path)
	return ok
}

func handleGCPSecurityPublicCAListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPublicCAPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPublicCALocation(project, "global"),
		gcpSecurityPublicCALocation(project, "us-central1"),
	}
	return respondGCPSecurityPublicCAList(w, "locations", items, pageSize, start, path)
}

func handleGCPSecurityPublicCAGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPublicCALocation(project, location))
	return true
}

func handleGCPSecurityPublicCACreateExternalAccountKey(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPSecurityPublicCAExternalAccountKeysCollectionPath(path)
	if !ok {
		return false
	}

	if location != "global" {
		respondGCPSecurityPublicCAFailedPrecondition(w, path, "location must be global")
		return true
	}

	body, valid := decodeGCPSecurityPublicCAJSONBody(w, r, path)
	if !valid {
		return true
	}

	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	if bodyParent := gcpSecurityPublicCAString(body, "parent"); bodyParent != "" && bodyParent != parent {
		respondGCPSecurityPublicCAInvalidArgument(w, path, "parent must match requested resource")
		return true
	}

	externalAccountKey, present := gcpSecurityPublicCAExternalAccountKeyFromBody(body)
	if !present {
		respondGCPSecurityPublicCAInvalidArgument(w, path, "externalAccountKey is required")
		return true
	}

	keyID := "eak-1"
	if providedKeyID := gcpSecurityPublicCAString(externalAccountKey, "keyId"); providedKeyID != "" {
		keyID = providedKeyID
	}
	if name := gcpSecurityPublicCAString(externalAccountKey, "name"); name != "" {
		nameProject, nameLocation, parsedKeyID, parsed := parseGCPSecurityPublicCAExternalAccountKeyName(name)
		if !parsed {
			respondGCPSecurityPublicCAInvalidArgument(w, path, "externalAccountKey.name is invalid")
			return true
		}
		if nameProject != project || nameLocation != location {
			respondGCPSecurityPublicCAInvalidArgument(w, path, "externalAccountKey.name must match parent")
			return true
		}
		keyID = parsedKeyID
	}

	respondJSON(w, http.StatusOK, gcpSecurityPublicCAExternalAccountKey(project, location, keyID))
	return true
}

func parseGCPSecurityPublicCAPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPSecurityPublicCAInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPSecurityPublicCAInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPSecurityPublicCAList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSecurityPublicCAInvalidArgument(w, path, "pageToken is out of range")
		return false
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

func decodeGCPSecurityPublicCAJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPSecurityPublicCAInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpSecurityPublicCAExternalAccountKeyFromBody(body map[string]any) (map[string]any, bool) {
	if body == nil {
		return nil, false
	}
	if nestedRaw, ok := body["externalAccountKey"]; ok {
		nested, _ := nestedRaw.(map[string]any)
		if nested == nil {
			return nil, false
		}
		return nested, true
	}
	if len(body) == 0 {
		return nil, false
	}
	if _, hasParent := body["parent"]; hasParent && len(body) == 1 {
		return nil, false
	}
	return body, true
}

func gcpSecurityPublicCAString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func parseGCPSecurityPublicCAExternalAccountKeysCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "externalAccountKeys" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPSecurityPublicCAExternalAccountKeyName(name string) (project, location, keyID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "externalAccountKeys" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	keyID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || keyID == "" {
		return "", "", "", false
	}
	return project, location, keyID, true
}

func gcpSecurityPublicCALocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Public CA " + location,
		"labels": map[string]string{
			"service": "security_publicca",
			"stage":   "emulated",
		},
	}
}

func gcpSecurityPublicCAExternalAccountKey(project, location, keyID string) map[string]any {
	return map[string]any{
		"name":      fmt.Sprintf("projects/%s/locations/%s/externalAccountKeys/%s", project, location, keyID),
		"keyId":     keyID,
		"b64MacKey": "c3RhY2t5YXJkLW1hYy1rZXk",
	}
}

func respondGCPSecurityPublicCAInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPSecurityPublicCAFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_security_publicca(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "security_publicca") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/locations/global/externalAccountKeys/eak-1",
			"keyId":     "eak-1",
			"b64MacKey": "c3RhY2t5YXJkLW1hYy1rZXk",
			"service":   "security_publicca",
			"provider":  providerGCP,
			"path":      path,
		})
		return true
	}
	return false
}
