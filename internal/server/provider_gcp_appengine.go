package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPAppEngineRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)

	// Route recognition for App Engine Admin v1 resources:
	// - /v1/apps/*
	// - /v1/apps/*/services/*
	// - /v1/apps/*/services/*/versions/*
	// - /v1/apps/*/services/*/versions/*/instances/*
	// - /v1/apps/*/domainMappings/*
	// - /v1/apps/*/authorizedDomains*
	// - /v1/apps/*/authorizedCertificates*
	// - /v1/apps/*/firewall/ingressRules*
	if path == "/gcp/v1/apps" {
		if r.Method == http.MethodPost {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		return false
	}
	if !strings.HasPrefix(path, "/gcp/v1/apps/") {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPAppEngineGetApplication(w, path) {
			return true
		}
		if handleGCPAppEngineListServices(w, r, path) {
			return true
		}
		if handleGCPAppEngineGetService(w, path) {
			return true
		}
		if handleGCPAppEngineListVersions(w, r, path) {
			return true
		}
		if handleGCPAppEngineGetVersion(w, r, path) {
			return true
		}
		if handleGCPAppEngineListInstances(w, r, path) {
			return true
		}
		if handleGCPAppEngineGetInstance(w, path) {
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

func handleGCPAppEngineGetApplication(w http.ResponseWriter, path string) bool {
	appID, ok := parseGCPAppEngineAppPath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpAppEngineApplication(appID))
	return true
}

func handleGCPAppEngineListServices(w http.ResponseWriter, r *http.Request, path string) bool {
	appID, ok := parseGCPAppEngineServicesCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPAppEnginePagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpAppEngineService(appID, "default"),
	}
	return respondGCPAppEngineList(w, "services", items, pageSize, start, path)
}

func handleGCPAppEngineGetService(w http.ResponseWriter, path string) bool {
	appID, serviceID, ok := parseGCPAppEngineServicePath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpAppEngineService(appID, serviceID))
	return true
}

func handleGCPAppEngineListVersions(w http.ResponseWriter, r *http.Request, path string) bool {
	appID, serviceID, ok := parseGCPAppEngineVersionsCollectionPath(path)
	if !ok {
		return false
	}

	if !isValidGCPAppEngineVersionView(r.URL.Query().Get("view")) {
		respondGCPAppEngineInvalidArgument(w, path, "view must be one of VERSION_VIEW_UNSPECIFIED, BASIC, FULL")
		return true
	}

	pageSize, start, valid := parseGCPAppEnginePagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpAppEngineVersion(appID, serviceID, "v1"),
	}
	return respondGCPAppEngineList(w, "versions", items, pageSize, start, path)
}

func handleGCPAppEngineGetVersion(w http.ResponseWriter, r *http.Request, path string) bool {
	appID, serviceID, versionID, ok := parseGCPAppEngineVersionPath(path)
	if !ok {
		return false
	}

	if !isValidGCPAppEngineVersionView(r.URL.Query().Get("view")) {
		respondGCPAppEngineInvalidArgument(w, path, "view must be one of VERSION_VIEW_UNSPECIFIED, BASIC, FULL")
		return true
	}

	respondJSON(w, http.StatusOK, gcpAppEngineVersion(appID, serviceID, versionID))
	return true
}

func handleGCPAppEngineListInstances(w http.ResponseWriter, r *http.Request, path string) bool {
	appID, serviceID, versionID, ok := parseGCPAppEngineInstancesCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPAppEnginePagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpAppEngineInstance(appID, serviceID, versionID, "i-1"),
	}
	return respondGCPAppEngineList(w, "instances", items, pageSize, start, path)
}

func handleGCPAppEngineGetInstance(w http.ResponseWriter, path string) bool {
	appID, serviceID, versionID, instanceID, ok := parseGCPAppEngineInstancePath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpAppEngineInstance(appID, serviceID, versionID, instanceID))
	return true
}

func parseGCPAppEngineAppPath(path string) (appID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "apps" {
		return "", false
	}
	appID = strings.TrimSpace(parts[3])
	if appID == "" {
		return "", false
	}
	return appID, true
}

func parseGCPAppEngineServicesCollectionPath(path string) (appID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "apps" || parts[4] != "services" {
		return "", false
	}
	appID = strings.TrimSpace(parts[3])
	if appID == "" {
		return "", false
	}
	return appID, true
}

func parseGCPAppEngineServicePath(path string) (appID, serviceID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "apps" || parts[4] != "services" {
		return "", "", false
	}
	appID = strings.TrimSpace(parts[3])
	serviceID = strings.TrimSpace(parts[5])
	if appID == "" || serviceID == "" {
		return "", "", false
	}
	return appID, serviceID, true
}

func parseGCPAppEngineVersionsCollectionPath(path string) (appID, serviceID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "apps" || parts[4] != "services" || parts[6] != "versions" {
		return "", "", false
	}
	appID = strings.TrimSpace(parts[3])
	serviceID = strings.TrimSpace(parts[5])
	if appID == "" || serviceID == "" {
		return "", "", false
	}
	return appID, serviceID, true
}

func parseGCPAppEngineVersionPath(path string) (appID, serviceID, versionID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "apps" || parts[4] != "services" || parts[6] != "versions" {
		return "", "", "", false
	}
	appID = strings.TrimSpace(parts[3])
	serviceID = strings.TrimSpace(parts[5])
	versionID = strings.TrimSpace(parts[7])
	if appID == "" || serviceID == "" || versionID == "" {
		return "", "", "", false
	}
	return appID, serviceID, versionID, true
}

func parseGCPAppEngineInstancesCollectionPath(path string) (appID, serviceID, versionID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "apps" || parts[4] != "services" || parts[6] != "versions" || parts[8] != "instances" {
		return "", "", "", false
	}
	appID = strings.TrimSpace(parts[3])
	serviceID = strings.TrimSpace(parts[5])
	versionID = strings.TrimSpace(parts[7])
	if appID == "" || serviceID == "" || versionID == "" {
		return "", "", "", false
	}
	return appID, serviceID, versionID, true
}

func parseGCPAppEngineInstancePath(path string) (appID, serviceID, versionID, instanceID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "apps" || parts[4] != "services" || parts[6] != "versions" || parts[8] != "instances" {
		return "", "", "", "", false
	}
	appID = strings.TrimSpace(parts[3])
	serviceID = strings.TrimSpace(parts[5])
	versionID = strings.TrimSpace(parts[7])
	instanceID = strings.TrimSpace(parts[9])
	if appID == "" || serviceID == "" || versionID == "" || instanceID == "" {
		return "", "", "", "", false
	}
	return appID, serviceID, versionID, instanceID, true
}

func parseGCPAppEnginePagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPAppEngineInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}

	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPAppEngineInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func isValidGCPAppEngineVersionView(raw string) bool {
	view := strings.TrimSpace(raw)
	if view == "" {
		return true
	}
	switch strings.ToUpper(view) {
	case "VERSION_VIEW_UNSPECIFIED", "BASIC", "FULL", "0", "1", "2":
		return true
	default:
		return false
	}
}

func respondGCPAppEngineList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPAppEngineInvalidArgument(w, path, "pageToken is out of range")
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

func gcpAppEngineApplication(appID string) map[string]any {
	return map[string]any{
		"name":          fmt.Sprintf("apps/%s", appID),
		"id":            appID,
		"locationId":    "us-central",
		"servingStatus": "SERVING",
	}
}

func gcpAppEngineService(appID, serviceID string) map[string]any {
	versionName := fmt.Sprintf("apps/%s/services/%s/versions/v1", appID, serviceID)
	return map[string]any{
		"name": fmt.Sprintf("apps/%s/services/%s", appID, serviceID),
		"id":   serviceID,
		"split": map[string]any{
			"allocations": map[string]any{
				versionName: 1,
			},
		},
	}
}

func gcpAppEngineVersion(appID, serviceID, versionID string) map[string]any {
	return map[string]any{
		"name":          fmt.Sprintf("apps/%s/services/%s/versions/%s", appID, serviceID, versionID),
		"id":            versionID,
		"runtime":       "go122",
		"servingStatus": "SERVING",
		"env":           "STANDARD",
	}
}

func gcpAppEngineInstance(appID, serviceID, versionID, instanceID string) map[string]any {
	return map[string]any{
		"name":         fmt.Sprintf("apps/%s/services/%s/versions/%s/instances/%s", appID, serviceID, versionID, instanceID),
		"id":           instanceID,
		"vmName":       fmt.Sprintf("instance-%s", instanceID),
		"availability": "RESIDENT",
	}
}

func respondGCPAppEngineInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
