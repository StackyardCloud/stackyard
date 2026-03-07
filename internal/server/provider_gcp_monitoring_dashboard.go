package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPMonitoringDashboardRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPMonitoringDashboardPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.monitoring.dashboard.v1.DashboardsService/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPMonitoringDashboardList(w, r, path) {
			return true
		}
		if handleGCPMonitoringDashboardGetProjectDashboard(w, path) {
			return true
		}
		if handleGCPMonitoringDashboardGetManagedDashboard(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPMonitoringDashboardCreate(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPMonitoringDashboardUpdate(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPMonitoringDashboardDelete(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPMonitoringDashboardPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.monitoring.dashboard.v1.DashboardsService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/v1/dashboards/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}

	return strings.Contains(path, "/dashboards")
}

func handleGCPMonitoringDashboardList(w http.ResponseWriter, r *http.Request, path string) bool {
	project, dashboardID, ok := parseGCPMonitoringDashboardProjectPath(path)
	if !ok || dashboardID != "" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringDashboardPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpMonitoringDashboard(project, "dashboard-1")}
	return respondGCPMonitoringDashboardList(w, items, pageSize, start, path)
}

func handleGCPMonitoringDashboardGetProjectDashboard(w http.ResponseWriter, path string) bool {
	project, dashboardID, ok := parseGCPMonitoringDashboardProjectPath(path)
	if !ok || dashboardID == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpMonitoringDashboard(project, dashboardID))
	return true
}

func handleGCPMonitoringDashboardGetManagedDashboard(w http.ResponseWriter, path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "dashboards" || strings.TrimSpace(parts[3]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":        "dashboards/" + parts[3],
		"displayName": "Managed Dashboard",
		"etag":        "managed-etag",
	})
	return true
}

func handleGCPMonitoringDashboardCreate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, dashboardID, ok := parseGCPMonitoringDashboardProjectPath(path)
	if !ok || dashboardID != "" {
		return false
	}
	body, valid := decodeGCPMonitoringDashboardJSONBody(w, r, path)
	if !valid {
		return true
	}
	dashboard := gcpMonitoringDashboardBodyMap(body, "dashboard")
	if len(dashboard) == 0 {
		respondGCPMonitoringDashboardInvalidArgument(w, path, "dashboard is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpMonitoringDashboard(project, "dashboard-1"))
	return true
}

func handleGCPMonitoringDashboardUpdate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, dashboardID, ok := parseGCPMonitoringDashboardProjectPath(path)
	if !ok || dashboardID == "" {
		return false
	}
	body, valid := decodeGCPMonitoringDashboardJSONBody(w, r, path)
	if !valid {
		return true
	}
	dashboard := gcpMonitoringDashboardBodyMap(body, "dashboard")
	if len(dashboard) == 0 {
		respondGCPMonitoringDashboardInvalidArgument(w, path, "dashboard is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpMonitoringDashboard(project, dashboardID))
	return true
}

func handleGCPMonitoringDashboardDelete(w http.ResponseWriter, path string) bool {
	_, dashboardID, ok := parseGCPMonitoringDashboardProjectPath(path)
	if !ok || dashboardID == "" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPMonitoringDashboardProjectPath(path string) (project, dashboardID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "dashboards" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", "", false
	}
	if len(parts) == 5 {
		return project, "", true
	}
	if len(parts) == 6 && strings.TrimSpace(parts[5]) != "" {
		return project, parts[5], true
	}
	return "", "", false
}

func parseGCPMonitoringDashboardPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPMonitoringDashboardInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPMonitoringDashboardInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPMonitoringDashboardList(w http.ResponseWriter, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPMonitoringDashboardInvalidArgument(w, path, "pageToken is out of range")
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
		"dashboards":    items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPMonitoringDashboardJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPMonitoringDashboardInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpMonitoringDashboardBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpMonitoringDashboard(project, dashboardID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/dashboards/%s", project, dashboardID),
		"displayName": "Stackyard Dashboard",
		"etag":        "stackyard-dashboard-etag",
		"labels": map[string]string{
			"env": "local",
		},
	}
}

func respondGCPMonitoringDashboardInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
