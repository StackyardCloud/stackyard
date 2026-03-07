package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPLoggingRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPLoggingPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.logging.v2.LoggingServiceV2/") ||
		strings.HasPrefix(path, "/gcp/google.logging.v2.ConfigServiceV2/") ||
		strings.HasPrefix(path, "/gcp/google.logging.v2.MetricsServiceV2/") {
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
		if handleGCPLoggingListLogs(w, r, path) {
			return true
		}
		if handleGCPLoggingListMonitoredResourceDescriptors(w, r, path) {
			return true
		}
		if handleGCPLoggingListSinks(w, r, path) {
			return true
		}
		if handleGCPLoggingGetSink(w, path) {
			return true
		}
		if handleGCPLoggingListExclusions(w, r, path) {
			return true
		}
		if handleGCPLoggingGetExclusion(w, path) {
			return true
		}
		if handleGCPLoggingListLogMetrics(w, r, path) {
			return true
		}
		if handleGCPLoggingGetLogMetric(w, path) {
			return true
		}
		if handleGCPLoggingListBuckets(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPLoggingWriteEntries(w, r, path) {
			return true
		}
		if handleGCPLoggingListEntries(w, r, path) {
			return true
		}
		if handleGCPLoggingTailEntries(w, r, path) {
			return true
		}
		if handleGCPLoggingCreateSink(w, r, path) {
			return true
		}
		if handleGCPLoggingCreateExclusion(w, r, path) {
			return true
		}
		if handleGCPLoggingCreateLogMetric(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPLoggingDeleteLog(w, path) {
			return true
		}
		if handleGCPLoggingDeleteSink(w, path) {
			return true
		}
		if handleGCPLoggingDeleteExclusion(w, path) {
			return true
		}
		if handleGCPLoggingDeleteLogMetric(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPLoggingPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.logging.v2.LoggingServiceV2/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.logging.v2.ConfigServiceV2/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.logging.v2.MetricsServiceV2/") {
		return true
	}

	if strings.HasPrefix(path, "/gcp/v2/entries:write") ||
		strings.HasPrefix(path, "/gcp/v2/entries:list") ||
		strings.HasPrefix(path, "/gcp/v2/entries:tail") {
		return true
	}

	isResourceScope := strings.HasPrefix(path, "/gcp/v2/projects/") ||
		strings.HasPrefix(path, "/gcp/v2/organizations/") ||
		strings.HasPrefix(path, "/gcp/v2/folders/") ||
		strings.HasPrefix(path, "/gcp/v2/billingAccounts/") ||
		strings.HasPrefix(path, "/gcp/v2/logScopes/")
	if !isResourceScope {
		return false
	}

	return strings.Contains(path, "/logs") ||
		strings.Contains(path, "/monitoredResourceDescriptors") ||
		strings.Contains(path, "/sinks") ||
		strings.Contains(path, "/exclusions") ||
		strings.Contains(path, "/metrics") ||
		strings.Contains(path, "/buckets")
}

func handleGCPLoggingWriteEntries(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v2/entries:write" {
		return false
	}
	body, valid := decodeGCPLoggingJSONBody(w, r, path)
	if !valid {
		return true
	}
	entries, _ := body["entries"].([]any)
	if len(entries) == 0 {
		respondGCPLoggingInvalidArgument(w, path, "entries is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"logEntryErrors": map[string]any{},
	})
	return true
}

func handleGCPLoggingListEntries(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v2/entries:list" {
		return false
	}
	body, valid := decodeGCPLoggingJSONBody(w, r, path)
	if !valid {
		return true
	}
	resourceNames, _ := body["resourceNames"].([]any)
	if len(resourceNames) == 0 {
		respondGCPLoggingInvalidArgument(w, path, "resourceNames is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"entries":       []any{gcpLoggingLogEntry("projects/stackyard", "stackyard%2Fapp")},
		"nextPageToken": "",
	})
	return true
}

func handleGCPLoggingTailEntries(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v2/entries:tail" {
		return false
	}
	if _, valid := decodeGCPLoggingJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"entries": []any{gcpLoggingLogEntry("projects/stackyard", "stackyard%2Fapp")},
	})
	return true
}

func handleGCPLoggingListLogs(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "logs" {
		return false
	}
	pageSize, start, valid := parseGCPLoggingPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{{"name": fmt.Sprintf("%s/logs/%s", scope, "stackyard%2Fapp")}}
	return respondGCPLoggingList(w, "logNames", items, pageSize, start, path)
}

func handleGCPLoggingDeleteLog(w http.ResponseWriter, path string) bool {
	_, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "logs" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPLoggingListMonitoredResourceDescriptors(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "monitoredResourceDescriptors" {
		return false
	}
	pageSize, start, valid := parseGCPLoggingPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpLoggingMonitoredResourceDescriptor(scope)}
	return respondGCPLoggingList(w, "resourceDescriptors", items, pageSize, start, path)
}

func handleGCPLoggingListSinks(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "sinks" {
		return false
	}
	pageSize, start, valid := parseGCPLoggingPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpLoggingSink(scope, "export-a")}
	return respondGCPLoggingList(w, "sinks", items, pageSize, start, path)
}

func handleGCPLoggingGetSink(w http.ResponseWriter, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "sinks" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpLoggingSink(scope, tail[1]))
	return true
}

func handleGCPLoggingCreateSink(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "sinks" {
		return false
	}
	body, valid := decodeGCPLoggingJSONBody(w, r, path)
	if !valid {
		return true
	}
	sink := gcpLoggingBodyMap(body, "sink")
	if len(sink) == 0 {
		respondGCPLoggingInvalidArgument(w, path, "sink is required")
		return true
	}
	sinkName := strings.TrimSpace(stringFromMap(sink, "name"))
	if sinkName == "" {
		respondGCPLoggingInvalidArgument(w, path, "sink.name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpLoggingSink(scope, sinkName))
	return true
}

func handleGCPLoggingDeleteSink(w http.ResponseWriter, path string) bool {
	_, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "sinks" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPLoggingListExclusions(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "exclusions" {
		return false
	}
	pageSize, start, valid := parseGCPLoggingPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpLoggingExclusion(scope, "exclude-debug")}
	return respondGCPLoggingList(w, "exclusions", items, pageSize, start, path)
}

func handleGCPLoggingGetExclusion(w http.ResponseWriter, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "exclusions" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpLoggingExclusion(scope, tail[1]))
	return true
}

func handleGCPLoggingCreateExclusion(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "exclusions" {
		return false
	}
	body, valid := decodeGCPLoggingJSONBody(w, r, path)
	if !valid {
		return true
	}
	exclusion := gcpLoggingBodyMap(body, "exclusion")
	if len(exclusion) == 0 {
		respondGCPLoggingInvalidArgument(w, path, "exclusion is required")
		return true
	}
	exclusionName := strings.TrimSpace(stringFromMap(exclusion, "name"))
	if exclusionName == "" {
		respondGCPLoggingInvalidArgument(w, path, "exclusion.name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpLoggingExclusion(scope, exclusionName))
	return true
}

func handleGCPLoggingDeleteExclusion(w http.ResponseWriter, path string) bool {
	_, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "exclusions" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPLoggingListLogMetrics(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "metrics" {
		return false
	}
	pageSize, start, valid := parseGCPLoggingPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpLoggingMetric(scope, "error_count")}
	return respondGCPLoggingList(w, "metrics", items, pageSize, start, path)
}

func handleGCPLoggingGetLogMetric(w http.ResponseWriter, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "metrics" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpLoggingMetric(scope, tail[1]))
	return true
}

func handleGCPLoggingCreateLogMetric(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "metrics" {
		return false
	}
	body, valid := decodeGCPLoggingJSONBody(w, r, path)
	if !valid {
		return true
	}
	metric := gcpLoggingBodyMap(body, "metric")
	if len(metric) == 0 {
		respondGCPLoggingInvalidArgument(w, path, "metric is required")
		return true
	}
	metricName := strings.TrimSpace(stringFromMap(metric, "name"))
	if metricName == "" {
		respondGCPLoggingInvalidArgument(w, path, "metric.name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpLoggingMetric(scope, metricName))
	return true
}

func handleGCPLoggingDeleteLogMetric(w http.ResponseWriter, path string) bool {
	_, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "metrics" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPLoggingListBuckets(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, tail, ok := parseGCPLoggingScopeTail(path)
	if !ok || len(tail) != 3 || tail[0] != "locations" || strings.TrimSpace(tail[1]) == "" || tail[2] != "buckets" {
		return false
	}
	pageSize, start, valid := parseGCPLoggingPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpLoggingBucket(scope, tail[1], "_Default")}
	return respondGCPLoggingList(w, "buckets", items, pageSize, start, path)
}

func parseGCPLoggingScopeTail(path string) (scope string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v2" {
		return "", nil, false
	}
	scopeType := strings.TrimSpace(parts[2])
	scopeID := strings.TrimSpace(parts[3])
	switch scopeType {
	case "projects", "organizations", "folders", "billingAccounts", "logScopes":
	default:
		return "", nil, false
	}
	if scopeID == "" {
		return "", nil, false
	}
	scope = scopeType + "/" + scopeID
	return scope, parts[4:], true
}

func parseGCPLoggingPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPLoggingInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPLoggingInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPLoggingList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPLoggingInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPLoggingJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPLoggingInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpLoggingBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpLoggingLogEntry(scope, logID string) map[string]any {
	return map[string]any{
		"logName":     fmt.Sprintf("%s/logs/%s", scope, logID),
		"textPayload": "stackyard logging apiv2 example entry",
		"severity":    "INFO",
	}
}

func gcpLoggingMonitoredResourceDescriptor(scope string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("%s/monitoredResourceDescriptors/global", scope),
		"type":        "global",
		"displayName": "Global",
	}
}

func gcpLoggingSink(scope, sinkID string) map[string]any {
	return map[string]any{
		"name":           sinkID,
		"destination":    "storage.googleapis.com/stackyard-logs",
		"filter":         "severity>=ERROR",
		"writerIdentity": fmt.Sprintf("serviceAccount:%s@sink.local", strings.ReplaceAll(scope, "/", "-")),
	}
}

func gcpLoggingExclusion(scope, exclusionID string) map[string]any {
	return map[string]any{
		"name":        exclusionID,
		"description": "stackyard exclusion",
		"filter":      "severity=DEBUG",
		"disabled":    false,
		"resource":    scope,
	}
}

func gcpLoggingMetric(scope, metricID string) map[string]any {
	return map[string]any{
		"name":        metricID,
		"description": "stackyard error counter",
		"filter":      "severity>=ERROR",
		"bucketName":  scope,
	}
}

func gcpLoggingBucket(scope, location, bucketID string) map[string]any {
	return map[string]any{
		"name":           fmt.Sprintf("%s/locations/%s/buckets/%s", scope, location, bucketID),
		"description":    "default logging bucket",
		"lifecycleState": "ACTIVE",
	}
}

func respondGCPLoggingInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
