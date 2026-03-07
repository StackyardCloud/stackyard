package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPMonitoringRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPMonitoringPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.monitoring.v3.AlertPolicyService/") ||
		strings.HasPrefix(path, "/gcp/google.monitoring.v3.GroupService/") ||
		strings.HasPrefix(path, "/gcp/google.monitoring.v3.MetricService/") ||
		strings.HasPrefix(path, "/gcp/google.monitoring.v3.NotificationChannelService/") ||
		strings.HasPrefix(path, "/gcp/google.monitoring.v3.ServiceMonitoringService/") ||
		strings.HasPrefix(path, "/gcp/google.monitoring.v3.SnoozeService/") ||
		strings.HasPrefix(path, "/gcp/google.monitoring.v3.UptimeCheckService/") ||
		strings.HasPrefix(path, "/gcp/google.monitoring.v3.QueryService/") {
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
		if handleGCPMonitoringListUptimeCheckIPs(w, r, path) {
			return true
		}
		if handleGCPMonitoringListAlertPolicies(w, r, path) {
			return true
		}
		if handleGCPMonitoringListGroups(w, r, path) {
			return true
		}
		if handleGCPMonitoringListGroupMembers(w, r, path) {
			return true
		}
		if handleGCPMonitoringListMetricDescriptors(w, r, path) {
			return true
		}
		if handleGCPMonitoringListMonitoredResourceDescriptors(w, r, path) {
			return true
		}
		if handleGCPMonitoringListNotificationChannelDescriptors(w, r, path) {
			return true
		}
		if handleGCPMonitoringListNotificationChannels(w, r, path) {
			return true
		}
		if handleGCPMonitoringListServices(w, r, path) {
			return true
		}
		if handleGCPMonitoringListServiceLevelObjectives(w, r, path) {
			return true
		}
		if handleGCPMonitoringListSnoozes(w, r, path) {
			return true
		}
		if handleGCPMonitoringListUptimeCheckConfigs(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPMonitoringCreateAlertPolicy(w, r, path) {
			return true
		}
		if handleGCPMonitoringQueryTimeSeries(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPMonitoringPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.monitoring.v3.AlertPolicyService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.monitoring.v3.GroupService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.monitoring.v3.MetricService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.monitoring.v3.NotificationChannelService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.monitoring.v3.ServiceMonitoringService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.monitoring.v3.SnoozeService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.monitoring.v3.UptimeCheckService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.monitoring.v3.QueryService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/v3/uptimeCheckIps") {
		return true
	}

	isProjectOrWorkspace := strings.HasPrefix(path, "/gcp/v3/projects/") || strings.HasPrefix(path, "/gcp/v3/workspaces/")
	if !isProjectOrWorkspace {
		return false
	}

	return strings.Contains(path, "/alertPolicies") ||
		strings.Contains(path, "/groups") ||
		strings.Contains(path, "/members") ||
		strings.Contains(path, "/metricDescriptors") ||
		strings.Contains(path, "/monitoredResourceDescriptors") ||
		strings.Contains(path, "/timeSeries") ||
		strings.Contains(path, ":query") ||
		strings.Contains(path, "/notificationChannelDescriptors") ||
		strings.Contains(path, "/notificationChannels") ||
		strings.Contains(path, "/services") ||
		strings.Contains(path, "/serviceLevelObjectives") ||
		strings.Contains(path, "/snoozes") ||
		strings.Contains(path, "/uptimeCheckConfigs")
}

func handleGCPMonitoringListUptimeCheckIPs(w http.ResponseWriter, r *http.Request, path string) bool {
	if !strings.HasPrefix(path, "/gcp/v3/uptimeCheckIps") {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{
		"region":    "USA",
		"location":  "us-central1",
		"ipAddress": "34.83.12.10",
	}}
	return respondGCPMonitoringList(w, "uptimeCheckIps", items, pageSize, start, path)
}

func handleGCPMonitoringListAlertPolicies(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || resourceType != "projects" || len(tail) != 1 || tail[0] != "alertPolicies" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpMonitoringAlertPolicy(resourceType, resourceID, "alert-1")}
	return respondGCPMonitoringList(w, "alertPolicies", items, pageSize, start, path)
}

func handleGCPMonitoringCreateAlertPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || resourceType != "projects" || len(tail) != 1 || tail[0] != "alertPolicies" {
		return false
	}
	body, valid := decodeGCPMonitoringJSONBody(w, r, path)
	if !valid {
		return true
	}
	alertPolicy := gcpMonitoringBodyMap(body, "alertPolicy")
	if len(alertPolicy) == 0 {
		respondGCPMonitoringInvalidArgument(w, path, "alertPolicy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpMonitoringAlertPolicy(resourceType, resourceID, "alert-1"))
	return true
}

func handleGCPMonitoringListGroups(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || resourceType != "projects" || len(tail) != 1 || tail[0] != "groups" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpMonitoringGroup(resourceID, "group-1")}
	return respondGCPMonitoringList(w, "group", items, pageSize, start, path)
}

func handleGCPMonitoringListGroupMembers(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || resourceType != "projects" || len(tail) != 3 || tail[0] != "groups" || strings.TrimSpace(tail[1]) == "" || tail[2] != "members" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{
		"resource": map[string]any{
			"type": "gce_instance",
			"labels": map[string]string{
				"project_id":  resourceID,
				"instance_id": "12345",
				"zone":        "us-central1-a",
			},
		},
	}}
	return respondGCPMonitoringList(w, "members", items, pageSize, start, path)
}

func handleGCPMonitoringListMetricDescriptors(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || resourceType != "projects" || len(tail) != 1 || tail[0] != "metricDescriptors" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{
		"name":        fmt.Sprintf("projects/%s/metricDescriptors/compute.googleapis.com/instance/cpu/utilization", resourceID),
		"type":        "compute.googleapis.com/instance/cpu/utilization",
		"metricKind":  "GAUGE",
		"valueType":   "DOUBLE",
		"displayName": "CPU Utilization",
	}}
	return respondGCPMonitoringList(w, "metricDescriptors", items, pageSize, start, path)
}

func handleGCPMonitoringListMonitoredResourceDescriptors(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || resourceType != "projects" || len(tail) != 1 || tail[0] != "monitoredResourceDescriptors" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{
		"name":        fmt.Sprintf("projects/%s/monitoredResourceDescriptors/gce_instance", resourceID),
		"type":        "gce_instance",
		"displayName": "GCE Instance",
	}}
	return respondGCPMonitoringList(w, "resourceDescriptors", items, pageSize, start, path)
}

func handleGCPMonitoringQueryTimeSeries(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || resourceType != "projects" || len(tail) != 1 {
		return false
	}
	segment, action, hasAction := strings.Cut(normalizeGCPMonitoringActionSegment(tail[0]), ":")
	if !hasAction || segment != "timeSeries" || action != "query" {
		return false
	}
	body, valid := decodeGCPMonitoringJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(stringFromMap(body, "query")) == "" {
		respondGCPMonitoringInvalidArgument(w, path, "query is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"timeSeriesData": []any{
			map[string]any{
				"labelValues": []any{},
				"pointData": []any{
					map[string]any{
						"values": []any{map[string]any{"doubleValue": 0.42}},
					},
				},
				"resource": fmt.Sprintf("projects/%s", resourceID),
			},
		},
	})
	return true
}

func handleGCPMonitoringListNotificationChannelDescriptors(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, _, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || resourceType != "projects" || len(tail) != 1 || tail[0] != "notificationChannelDescriptors" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{
		"name":        "notificationChannelDescriptors/email",
		"type":        "email",
		"displayName": "Email",
	}}
	return respondGCPMonitoringList(w, "channelDescriptors", items, pageSize, start, path)
}

func handleGCPMonitoringListNotificationChannels(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || resourceType != "projects" || len(tail) != 1 || tail[0] != "notificationChannels" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{
		"name":        fmt.Sprintf("projects/%s/notificationChannels/channel-1", resourceID),
		"type":        "email",
		"displayName": "stackyard-email",
		"enabled":     true,
	}}
	return respondGCPMonitoringList(w, "notificationChannels", items, pageSize, start, path)
}

func handleGCPMonitoringListServices(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || len(tail) != 1 || tail[0] != "services" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{
		"name":        fmt.Sprintf("%s/%s/services/service-a", resourceType, resourceID),
		"displayName": "service-a",
	}}
	return respondGCPMonitoringList(w, "services", items, pageSize, start, path)
}

func handleGCPMonitoringListServiceLevelObjectives(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || len(tail) != 3 || tail[0] != "services" || strings.TrimSpace(tail[1]) == "" || tail[2] != "serviceLevelObjectives" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{
		"name":        fmt.Sprintf("%s/%s/services/%s/serviceLevelObjectives/slo-1", resourceType, resourceID, tail[1]),
		"displayName": "slo-1",
	}}
	return respondGCPMonitoringList(w, "serviceLevelObjectives", items, pageSize, start, path)
}

func handleGCPMonitoringListSnoozes(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || resourceType != "projects" || len(tail) != 1 || tail[0] != "snoozes" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{
		"name":     fmt.Sprintf("projects/%s/snoozes/snooze-1", resourceID),
		"criteria": map[string]any{},
	}}
	return respondGCPMonitoringList(w, "snoozes", items, pageSize, start, path)
}

func handleGCPMonitoringListUptimeCheckConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	resourceType, resourceID, tail, ok := parseGCPMonitoringResourceTail(path)
	if !ok || resourceType != "projects" || len(tail) != 1 || tail[0] != "uptimeCheckConfigs" {
		return false
	}
	pageSize, start, valid := parseGCPMonitoringPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{
		"name":        fmt.Sprintf("projects/%s/uptimeCheckConfigs/uptime-1", resourceID),
		"displayName": "uptime-1",
	}}
	return respondGCPMonitoringList(w, "uptimeCheckConfigs", items, pageSize, start, path)
}

func parseGCPMonitoringResourceTail(path string) (resourceType, resourceID string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v3" {
		return "", "", nil, false
	}
	resourceType = strings.TrimSpace(parts[2])
	resourceID = strings.TrimSpace(parts[3])
	if (resourceType != "projects" && resourceType != "workspaces") || resourceID == "" {
		return "", "", nil, false
	}
	return resourceType, resourceID, parts[4:], true
}

func parseGCPMonitoringPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPMonitoringInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPMonitoringInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPMonitoringList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPMonitoringInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPMonitoringJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPMonitoringInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpMonitoringBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func normalizeGCPMonitoringActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func gcpMonitoringAlertPolicy(resourceType, resourceID, alertID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("%s/%s/alertPolicies/%s", resourceType, resourceID, alertID),
		"displayName": "stackyard alert policy",
		"enabled":     true,
	}
}

func gcpMonitoringGroup(projectID, groupID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/groups/%s", projectID, groupID),
		"displayName": "stackyard group",
	}
}

func respondGCPMonitoringInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
