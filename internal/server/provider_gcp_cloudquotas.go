package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPCloudQuotasRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v1/") {
		return false
	}
	if !isGCPCloudQuotasPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPCloudQuotasListQuotaInfos(w, r, path) {
			return true
		}
		if handleGCPCloudQuotasGetQuotaInfo(w, path) {
			return true
		}
		if handleGCPCloudQuotasListQuotaPreferences(w, r, path) {
			return true
		}
		if handleGCPCloudQuotasGetQuotaPreference(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPCloudQuotasCreateQuotaPreference(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPCloudQuotasUpdateQuotaPreference(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPCloudQuotasPath(path string) bool {
	_, _, _, tail, ok := parseGCPCloudQuotasLocationTail(path)
	if !ok || len(tail) == 0 {
		return false
	}
	return isGCPCloudQuotasQuotaInfosCollectionTail(tail) ||
		isGCPCloudQuotasQuotaInfoTail(tail) ||
		isGCPCloudQuotasQuotaPreferencesCollectionTail(tail) ||
		isGCPCloudQuotasQuotaPreferenceTail(tail)
}

func handleGCPCloudQuotasListQuotaInfos(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPCloudQuotasLocationTail(path)
	if !ok || !isGCPCloudQuotasQuotaInfosCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPCloudQuotasPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudQuotasQuotaInfo(scope, scopeID, location, tail[1], "CpusPerProjectPerRegion")}
	return respondGCPCloudQuotasList(w, "quotaInfos", items, pageSize, start, path)
}

func handleGCPCloudQuotasGetQuotaInfo(w http.ResponseWriter, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPCloudQuotasLocationTail(path)
	if !ok || !isGCPCloudQuotasQuotaInfoTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudQuotasQuotaInfo(scope, scopeID, location, tail[1], tail[3]))
	return true
}

func handleGCPCloudQuotasListQuotaPreferences(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPCloudQuotasLocationTail(path)
	if !ok || !isGCPCloudQuotasQuotaPreferencesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPCloudQuotasPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudQuotasQuotaPreference(scope, scopeID, location, "team-config", "compute.googleapis.com", "CpusPerProjectPerRegion", 16)}
	return respondGCPCloudQuotasList(w, "quotaPreferences", items, pageSize, start, path)
}

func handleGCPCloudQuotasGetQuotaPreference(w http.ResponseWriter, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPCloudQuotasLocationTail(path)
	if !ok || !isGCPCloudQuotasQuotaPreferenceTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudQuotasQuotaPreference(scope, scopeID, location, tail[1], "compute.googleapis.com", "CpusPerProjectPerRegion", 16))
	return true
}

func handleGCPCloudQuotasCreateQuotaPreference(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPCloudQuotasLocationTail(path)
	if !ok || !isGCPCloudQuotasQuotaPreferencesCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPCloudQuotasJSONBody(w, r, path)
	if !valid {
		return true
	}
	quotaPreference := gcpCloudQuotasBodyMap(body, "quotaPreference")
	if len(quotaPreference) == 0 {
		respondGCPCloudQuotasInvalidArgument(w, path, "quotaPreference is required")
		return true
	}
	service := strings.TrimSpace(gcpCloudQuotasString(quotaPreference, "service"))
	quotaID := strings.TrimSpace(gcpCloudQuotasString(quotaPreference, "quotaId"))
	if service == "" || quotaID == "" {
		respondGCPCloudQuotasInvalidArgument(w, path, "quotaPreference.service and quotaPreference.quotaId are required")
		return true
	}
	preferredValue, err := gcpCloudQuotasPreferredValue(quotaPreference)
	if err != nil {
		respondGCPCloudQuotasInvalidArgument(w, path, err.Error())
		return true
	}
	quotaPreferenceID := strings.TrimSpace(r.URL.Query().Get("quotaPreferenceId"))
	if quotaPreferenceID == "" {
		quotaPreferenceID = "team-config"
	}
	respondJSON(w, http.StatusOK, gcpCloudQuotasQuotaPreference(scope, scopeID, location, quotaPreferenceID, service, quotaID, preferredValue))
	return true
}

func handleGCPCloudQuotasUpdateQuotaPreference(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPCloudQuotasLocationTail(path)
	if !ok || !isGCPCloudQuotasQuotaPreferenceTail(tail) {
		return false
	}
	body, valid := decodeGCPCloudQuotasJSONBody(w, r, path)
	if !valid {
		return true
	}
	quotaPreference := gcpCloudQuotasBodyMap(body, "quotaPreference")
	if len(quotaPreference) == 0 {
		respondGCPCloudQuotasInvalidArgument(w, path, "quotaPreference is required")
		return true
	}
	name := strings.TrimSpace(gcpCloudQuotasString(quotaPreference, "name"))
	expectedSuffix := "/quotaPreferences/" + tail[1]
	if name != "" && !strings.HasSuffix(name, expectedSuffix) {
		respondGCPCloudQuotasInvalidArgument(w, path, "quotaPreference.name must match the requested resource")
		return true
	}
	service := strings.TrimSpace(gcpCloudQuotasString(quotaPreference, "service"))
	quotaID := strings.TrimSpace(gcpCloudQuotasString(quotaPreference, "quotaId"))
	if service == "" || quotaID == "" {
		respondGCPCloudQuotasInvalidArgument(w, path, "quotaPreference.service and quotaPreference.quotaId are required")
		return true
	}
	preferredValue, err := gcpCloudQuotasPreferredValue(quotaPreference)
	if err != nil {
		respondGCPCloudQuotasInvalidArgument(w, path, err.Error())
		return true
	}
	respondJSON(w, http.StatusOK, gcpCloudQuotasQuotaPreference(scope, scopeID, location, tail[1], service, quotaID, preferredValue))
	return true
}

func parseGCPCloudQuotasLocationTail(path string) (scope, scopeID, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[4] != "locations" {
		return "", "", "", nil, false
	}
	scope = strings.TrimSpace(parts[2])
	if scope != "projects" && scope != "folders" && scope != "organizations" {
		return "", "", "", nil, false
	}
	scopeID = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if scopeID == "" || location == "" {
		return "", "", "", nil, false
	}
	return scope, scopeID, location, parts[6:], true
}

func isGCPCloudQuotasQuotaInfosCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "services" && strings.TrimSpace(tail[1]) != "" && tail[2] == "quotaInfos"
}

func isGCPCloudQuotasQuotaInfoTail(tail []string) bool {
	return len(tail) == 4 && tail[0] == "services" && strings.TrimSpace(tail[1]) != "" && tail[2] == "quotaInfos" && strings.TrimSpace(tail[3]) != ""
}

func isGCPCloudQuotasQuotaPreferencesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "quotaPreferences"
}

func isGCPCloudQuotasQuotaPreferenceTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "quotaPreferences" && strings.TrimSpace(tail[1]) != ""
}

func parseGCPCloudQuotasPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPCloudQuotasInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	start = 0
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPCloudQuotasInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPCloudQuotasList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPCloudQuotasInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPCloudQuotasJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPCloudQuotasInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpCloudQuotasBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpCloudQuotasString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func gcpCloudQuotasPreferredValue(quotaPreference map[string]any) (int, error) {
	quotaConfig, _ := quotaPreference["quotaConfig"].(map[string]any)
	if len(quotaConfig) == 0 {
		return 0, fmt.Errorf("quotaPreference.quotaConfig is required")
	}
	value, ok := quotaConfig["preferredValue"]
	if !ok {
		return 0, fmt.Errorf("quotaPreference.quotaConfig.preferredValue is required")
	}
	switch typed := value.(type) {
	case float64:
		if typed <= 0 || typed != float64(int(typed)) {
			return 0, fmt.Errorf("quotaPreference.quotaConfig.preferredValue must be a positive integer")
		}
		return int(typed), nil
	case int:
		if typed <= 0 {
			return 0, fmt.Errorf("quotaPreference.quotaConfig.preferredValue must be a positive integer")
		}
		return typed, nil
	default:
		return 0, fmt.Errorf("quotaPreference.quotaConfig.preferredValue must be a positive integer")
	}
}

func gcpCloudQuotasQuotaInfo(scope, scopeID, location, service, quotaID string) map[string]any {
	return map[string]any{
		"name":               fmt.Sprintf("%s/%s/locations/%s/services/%s/quotaInfos/%s", scope, scopeID, location, service, quotaID),
		"quotaId":            quotaID,
		"service":            service,
		"isPrecise":          true,
		"refreshInterval":    "60s",
		"containerType":      "PROJECT",
		"containerDisplayId": scopeID,
	}
}

func gcpCloudQuotasQuotaPreference(scope, scopeID, location, preferenceID, service, quotaID string, preferredValue int) map[string]any {
	return map[string]any{
		"name":    fmt.Sprintf("%s/%s/locations/%s/quotaPreferences/%s", scope, scopeID, location, preferenceID),
		"service": service,
		"quotaId": quotaID,
		"quotaConfig": map[string]any{
			"preferredValue": preferredValue,
		},
		"dimensions": map[string]string{
			"region": "us-central1",
		},
		"justification": "stackyard staged quota request",
		"contactEmail":  "stackyard@example.com",
	}
}

func respondGCPCloudQuotasInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
