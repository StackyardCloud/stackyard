package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPAPIGatewayRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPAPIGatewayPath(path, hasGCPAPIGatewayHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPAPIGatewayListApis(w, r, path) {
			return true
		}
		if handleGCPAPIGatewayGetAPI(w, path) {
			return true
		}
		if handleGCPAPIGatewayListAPIConfigs(w, r, path) {
			return true
		}
		if handleGCPAPIGatewayGetAPIConfig(w, path) {
			return true
		}
		if handleGCPAPIGatewayListGateways(w, r, path) {
			return true
		}
		if handleGCPAPIGatewayGetGateway(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost, http.MethodPatch, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPAPIGatewayHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "apigateway", "api-gateway", "api_gateway", "apigateway-apiv1", "apigateway_apiv1":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-apigateway-apiv1")
}

func isGCPAPIGatewayPath(path string, includeHint bool) bool {
	if _, ok := parseGCPAPIGatewayParentCollectionPath(path, "apis"); ok {
		return true
	}
	if _, _, ok := parseGCPAPIGatewayResourcePath(path, "apis"); ok {
		return true
	}
	if _, _, ok := parseGCPAPIGatewayAPIConfigCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPAPIGatewayAPIConfigResourcePath(path); ok {
		return true
	}
	if _, ok := parseGCPAPIGatewayParentCollectionPath(path, "gateways"); ok {
		return includeHint
	}
	_, _, ok := parseGCPAPIGatewayResourcePath(path, "gateways")
	return includeHint && ok
}

func handleGCPAPIGatewayListApis(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPAPIGatewayParentCollectionPath(path, "apis")
	if !ok {
		return false
	}

	pageSize, start, ok := parseGCPAPIGatewayPagination(w, r, path)
	if !ok {
		return true
	}

	items := []map[string]any{
		gcpAPIGatewayAPI(parent, "team-api"),
	}
	return respondGCPAPIGatewayList(w, items, "apis", start, pageSize, path)
}

func handleGCPAPIGatewayGetAPI(w http.ResponseWriter, path string) bool {
	parent, apiID, ok := parseGCPAPIGatewayResourcePath(path, "apis")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpAPIGatewayAPI(parent, apiID))
	return true
}

func handleGCPAPIGatewayListAPIConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, apiID, ok := parseGCPAPIGatewayAPIConfigCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, ok := parseGCPAPIGatewayPagination(w, r, path)
	if !ok {
		return true
	}

	items := []map[string]any{
		gcpAPIGatewayAPIConfig(parent, apiID, "team-config"),
	}
	return respondGCPAPIGatewayList(w, items, "apiConfigs", start, pageSize, path)
}

func handleGCPAPIGatewayGetAPIConfig(w http.ResponseWriter, path string) bool {
	parent, apiID, configID, ok := parseGCPAPIGatewayAPIConfigResourcePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpAPIGatewayAPIConfig(parent, apiID, configID))
	return true
}

func handleGCPAPIGatewayListGateways(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPAPIGatewayParentCollectionPath(path, "gateways")
	if !ok {
		return false
	}

	pageSize, start, ok := parseGCPAPIGatewayPagination(w, r, path)
	if !ok {
		return true
	}

	items := []map[string]any{
		gcpAPIGatewayGateway(parent, "team-gateway"),
	}
	return respondGCPAPIGatewayList(w, items, "gateways", start, pageSize, path)
}

func handleGCPAPIGatewayGetGateway(w http.ResponseWriter, path string) bool {
	parent, gatewayID, ok := parseGCPAPIGatewayResourcePath(path, "gateways")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpAPIGatewayGateway(parent, gatewayID))
	return true
}

func parseGCPAPIGatewayParentCollectionPath(path, collection string) (parent string, ok bool) {
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

func parseGCPAPIGatewayResourcePath(path, collection string) (parent, resourceID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 {
		return "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != collection {
		return "", "", false
	}
	if strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(parts[5]) == "" || strings.TrimSpace(parts[7]) == "" {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[3], parts[5]), parts[7], true
}

func parseGCPAPIGatewayAPIConfigCollectionPath(path string) (parent, apiID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 {
		return "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "apis" || parts[8] != "configs" {
		return "", "", false
	}
	if strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(parts[5]) == "" || strings.TrimSpace(parts[7]) == "" {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[3], parts[5]), parts[7], true
}

func parseGCPAPIGatewayAPIConfigResourcePath(path string) (parent, apiID, configID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 {
		return "", "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "apis" || parts[8] != "configs" {
		return "", "", "", false
	}
	if strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(parts[5]) == "" || strings.TrimSpace(parts[7]) == "" || strings.TrimSpace(parts[9]) == "" {
		return "", "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[3], parts[5]), parts[7], parts[9], true
}

func parseGCPAPIGatewayPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPAPIGatewayInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPAPIGatewayInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPAPIGatewayList(w http.ResponseWriter, items []map[string]any, key string, start, pageSize int, path string) bool {
	if start > len(items) {
		respondGCPAPIGatewayInvalidArgument(w, path, "pageToken is out of range")
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

func gcpAPIGatewayAPI(parent, apiID string) map[string]any {
	return map[string]any{
		"name":           fmt.Sprintf("%s/apis/%s", parent, apiID),
		"displayName":    "Team API",
		"state":          "ACTIVE",
		"managedService": "team-api.apigateway.stackyard.local",
	}
}

func gcpAPIGatewayAPIConfig(parent, apiID, configID string) map[string]any {
	return map[string]any{
		"name":                  fmt.Sprintf("%s/apis/%s/configs/%s", parent, apiID, configID),
		"displayName":           "Team Config",
		"state":                 "ACTIVE",
		"gatewayServiceAccount": "apigateway-service-account@stackyard.iam.gserviceaccount.com",
	}
}

func gcpAPIGatewayGateway(parent, gatewayID string) map[string]any {
	return map[string]any{
		"name":            fmt.Sprintf("%s/gateways/%s", parent, gatewayID),
		"displayName":     "Team Gateway",
		"state":           "ACTIVE",
		"defaultHostname": fmt.Sprintf("%s-%s.gateway.dev", strings.ReplaceAll(strings.TrimPrefix(parent, "projects/"), "/locations/", "-"), gatewayID),
	}
}

func respondGCPAPIGatewayInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
