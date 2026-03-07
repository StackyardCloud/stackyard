package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	gcpTraceGRPCBatchWriteSpansPath = "/gcp/google.devtools.cloudtrace.v2.TraceService/BatchWriteSpans"
	gcpTraceGRPCCreateSpanPath      = "/gcp/google.devtools.cloudtrace.v2.TraceService/CreateSpan"
)

var (
	gcpTraceReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpTraceProjectRegex  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,62}$`)
	gcpTraceTraceIDRegex  = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)
	gcpTraceSpanIDRegex   = regexp.MustCompile(`^[a-fA-F0-9]{16}$`)
	gcpTraceSpanKindSet   = map[string]struct{}{
		"SPAN_KIND_UNSPECIFIED": {},
		"INTERNAL":              {},
		"SERVER":                {},
		"CLIENT":                {},
		"PRODUCER":              {},
		"CONSUMER":              {},
		"SPAN_KIND_INTERNAL":    {},
		"SPAN_KIND_SERVER":      {},
		"SPAN_KIND_CLIENT":      {},
		"SPAN_KIND_PRODUCER":    {},
		"SPAN_KIND_CONSUMER":    {},
	}
)

type gcpTraceNormalizedSpan struct {
	Name        string
	Project     string
	TraceID     string
	SpanID      string
	ParentSpan  string
	DisplayName string
	StartTime   time.Time
	EndTime     time.Time
	SpanKind    string
	SameProcess bool
}

type gcpTraceSpanInput struct {
	Name        string
	SpanID      string
	ParentSpan  string
	DisplayName string
	StartTime   time.Time
	EndTime     time.Time
	HasStart    bool
	HasEnd      bool
	SpanKind    string
	SameProcess *bool
}

func (s *Server) handleGCPTraceRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_trace(w, r) {
		return true
	}

	path := normalizeGCPTracePath(rawRequestPath(r))
	if !isGCPTracePath(path, hasGCPTraceHint(r)) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	body, valid := decodeGCPTraceJSONBody(w, r, path)
	if !valid {
		return true
	}

	if handleGCPTraceBatchWriteSpans(w, path, body) {
		return true
	}
	if handleGCPTraceCreateSpan(w, path, body) {
		return true
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func normalizeGCPTracePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPTraceHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "trace",
		"trace-apiv2",
		"trace_apiv2",
		"stackdriver-trace",
		"stackdriver_trace",
		"cloudtrace",
		"cloud-trace",
		"gcp-trace":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-trace-apiv2") || strings.Contains(ua, "cloud.google.com/go/trace/apiv2")
}

func isGCPTracePath(path string, includeHint bool) bool {
	if path == gcpTraceGRPCBatchWriteSpansPath || path == gcpTraceGRPCCreateSpanPath {
		return true
	}
	if _, ok := parseGCPTraceBatchWritePath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPTraceSpanPath(path); ok {
		return true
	}
	return includeHint && strings.HasPrefix(path, "/gcp/v2/projects/") && strings.Contains(path, "/traces")
}

func handleGCPTraceBatchWriteSpans(w http.ResponseWriter, path string, body map[string]any) bool {
	expectedProject := ""
	if path == gcpTraceGRPCBatchWriteSpansPath {
		name := strings.TrimSpace(gcpTraceString(body, "name"))
		project, ok := parseGCPTraceProjectName(name)
		if !ok {
			respondGCPTraceInvalidArgument(w, path, "name is required and must be projects/{project}")
			return true
		}
		expectedProject = project
	} else {
		project, ok := parseGCPTraceBatchWritePath(path)
		if !ok {
			return false
		}
		expectedProject = project
		if bodyName := strings.TrimSpace(gcpTraceString(body, "name")); bodyName != "" && bodyName != "projects/"+project {
			respondGCPTraceInvalidArgument(w, path, "name must match path project")
			return true
		}
	}

	if !isGCPTraceValidProjectID(expectedProject) {
		respondGCPTraceInvalidArgument(w, path, "project is invalid")
		return true
	}
	if isGCPTraceMissingProject(expectedProject) {
		respondGCPTraceNotFound(w, path, "project not found")
		return true
	}

	spans, ok := body["spans"].([]any)
	if !ok || len(spans) == 0 {
		respondGCPTraceInvalidArgument(w, path, "spans is required")
		return true
	}

	for i, rawSpan := range spans {
		spanMap, ok := rawSpan.(map[string]any)
		if !ok {
			respondGCPTraceInvalidArgument(w, path, fmt.Sprintf("spans[%d] must be an object", i))
			return true
		}
		input, ok := gcpTraceSpanInputFromMap(spanMap)
		if !ok {
			respondGCPTraceInvalidArgument(w, path, fmt.Sprintf("spans[%d] must be an object", i))
			return true
		}
		_, code, message := gcpTraceValidateAndNormalizeSpanInput(input, expectedProject, "", "")
		if code != "" {
			gcpTraceRespondValidationError(w, path, code, message)
			return true
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTraceCreateSpan(w http.ResponseWriter, path string, body map[string]any) bool {
	expectedProject := ""
	expectedTraceID := ""
	expectedSpanID := ""
	if path != gcpTraceGRPCCreateSpanPath {
		project, traceID, spanID, ok := parseGCPTraceSpanPath(path)
		if !ok {
			return false
		}
		expectedProject = project
		expectedTraceID = traceID
		expectedSpanID = spanID
	}

	input, ok := gcpTraceSpanInputFromMap(body)
	if !ok {
		respondGCPTraceInvalidArgument(w, path, "span body is required")
		return true
	}

	if expectedProject != "" && strings.TrimSpace(input.Name) == "" {
		input.Name = gcpTraceSpanName(expectedProject, expectedTraceID, expectedSpanID)
	}
	normalized, code, message := gcpTraceValidateAndNormalizeSpanInput(input, expectedProject, expectedTraceID, expectedSpanID)
	if code != "" {
		gcpTraceRespondValidationError(w, path, code, message)
		return true
	}
	if isGCPTraceMissingProject(normalized.Project) {
		respondGCPTraceNotFound(w, path, "project not found")
		return true
	}

	respondJSON(w, http.StatusOK, gcpTraceSpanJSONFixture(normalized))
	return true
}

func gcpTraceSpanInputFromMap(body map[string]any) (gcpTraceSpanInput, bool) {
	if len(body) == 0 {
		return gcpTraceSpanInput{}, false
	}
	start, hasStart := gcpTraceParseTimestamp(gcpTraceString(body, "startTime", "start_time"))
	end, hasEnd := gcpTraceParseTimestamp(gcpTraceString(body, "endTime", "end_time"))
	displayName := strings.TrimSpace(gcpTraceString(gcpTraceBodyMap(body, "displayName", "display_name"), "value"))
	sameProcess := gcpTraceOptionalBool(body, "sameProcessAsParentSpan", "same_process_as_parent_span")

	return gcpTraceSpanInput{
		Name:        strings.TrimSpace(gcpTraceString(body, "name")),
		SpanID:      strings.TrimSpace(gcpTraceString(body, "spanId", "span_id")),
		ParentSpan:  strings.TrimSpace(gcpTraceString(body, "parentSpanId", "parent_span_id")),
		DisplayName: displayName,
		StartTime:   start,
		EndTime:     end,
		HasStart:    hasStart,
		HasEnd:      hasEnd,
		SpanKind:    strings.TrimSpace(gcpTraceString(body, "spanKind", "span_kind")),
		SameProcess: sameProcess,
	}, true
}

func gcpTraceValidateAndNormalizeSpanInput(input gcpTraceSpanInput, expectedProject, expectedTraceID, expectedSpanID string) (gcpTraceNormalizedSpan, string, string) {
	project, traceID, spanIDFromName, ok := parseGCPTraceSpanName(input.Name)
	if !ok {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "name is required and must match projects/{project}/traces/{trace}/spans/{span}"
	}
	if expectedProject != "" && project != expectedProject {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "name project must match path project"
	}
	if expectedTraceID != "" && !strings.EqualFold(traceID, expectedTraceID) {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "name trace must match path trace"
	}
	if expectedSpanID != "" && !strings.EqualFold(spanIDFromName, expectedSpanID) {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "name span must match path span"
	}
	if !isGCPTraceValidProjectID(project) {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "project is invalid"
	}
	if !isGCPTraceValidTraceID(traceID) {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "trace id is invalid"
	}
	if !isGCPTraceValidSpanID(spanIDFromName) {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "span id is invalid"
	}

	spanIDField := strings.TrimSpace(input.SpanID)
	if spanIDField == "" {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "spanId is required"
	}
	if !isGCPTraceValidSpanID(spanIDField) {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "spanId is invalid"
	}
	if !strings.EqualFold(spanIDField, spanIDFromName) {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "spanId must match name span id"
	}

	parentSpan := strings.TrimSpace(input.ParentSpan)
	if parentSpan != "" && !isGCPTraceValidSpanID(parentSpan) {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "parentSpanId is invalid"
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "displayName.value is required"
	}
	if len([]byte(displayName)) > 128 {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "displayName.value must be <= 128 bytes"
	}

	if !input.HasStart {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "startTime is required"
	}
	if !input.HasEnd {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "endTime is required"
	}
	if input.EndTime.Before(input.StartTime) {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "endTime must be >= startTime"
	}
	if input.EndTime.Sub(input.StartTime) > 24*time.Hour {
		return gcpTraceNormalizedSpan{}, "FailedPrecondition", "span duration exceeds staged limit"
	}

	spanKind := gcpTraceNormalizeSpanKind(input.SpanKind)
	if _, ok := gcpTraceSpanKindSet[spanKind]; !ok {
		return gcpTraceNormalizedSpan{}, "InvalidArgument", "spanKind is invalid"
	}
	if spanKind == "SPAN_KIND_UNSPECIFIED" {
		spanKind = "SPAN_KIND_SERVER"
	}

	sameProcess := true
	if input.SameProcess != nil {
		sameProcess = *input.SameProcess
	}

	return gcpTraceNormalizedSpan{
		Name:        gcpTraceSpanName(project, strings.ToLower(traceID), strings.ToLower(spanIDFromName)),
		Project:     project,
		TraceID:     strings.ToLower(traceID),
		SpanID:      strings.ToLower(spanIDFromName),
		ParentSpan:  strings.ToLower(parentSpan),
		DisplayName: displayName,
		StartTime:   input.StartTime.UTC(),
		EndTime:     input.EndTime.UTC(),
		SpanKind:    spanKind,
		SameProcess: sameProcess,
	}, "", ""
}

func gcpTraceSpanJSONFixture(span gcpTraceNormalizedSpan) map[string]any {
	out := map[string]any{
		"name":     span.Name,
		"spanId":   span.SpanID,
		"spanKind": span.SpanKind,
		"displayName": map[string]any{
			"value": span.DisplayName,
		},
		"startTime":               span.StartTime.Format(time.RFC3339Nano),
		"endTime":                 span.EndTime.Format(time.RFC3339Nano),
		"sameProcessAsParentSpan": span.SameProcess,
		"attributes": map[string]any{
			"attributeMap": map[string]any{
				"stackyard.example": map[string]any{
					"stringValue": map[string]any{
						"value": "trace",
					},
				},
			},
			"droppedAttributesCount": 0,
		},
	}
	if span.ParentSpan != "" {
		out["parentSpanId"] = span.ParentSpan
	}
	return out
}

func parseGCPTraceBatchWritePath(path string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "traces:batchWrite" {
		return "", false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", false
	}
	return project, true
}

func parseGCPTraceSpanPath(path string) (project, traceID, spanID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "traces" || parts[6] != "spans" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	traceID = strings.TrimSpace(parts[5])
	spanID = strings.TrimSpace(parts[7])
	if project == "" || traceID == "" || spanID == "" {
		return "", "", "", false
	}
	return project, traceID, spanID, true
}

func parseGCPTraceProjectName(name string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 2 || parts[0] != "projects" {
		return "", false
	}
	project = strings.TrimSpace(parts[1])
	if project == "" {
		return "", false
	}
	return project, true
}

func parseGCPTraceSpanName(name string) (project, traceID, spanID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "traces" || parts[4] != "spans" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	traceID = strings.TrimSpace(parts[3])
	spanID = strings.TrimSpace(parts[5])
	if project == "" || traceID == "" || spanID == "" {
		return "", "", "", false
	}
	return project, traceID, spanID, true
}

func gcpTraceSpanName(project, traceID, spanID string) string {
	return fmt.Sprintf("projects/%s/traces/%s/spans/%s", project, traceID, spanID)
}

func isGCPTraceValidProjectID(project string) bool {
	return gcpTraceProjectRegex.MatchString(strings.TrimSpace(project))
}

func isGCPTraceValidTraceID(traceID string) bool {
	value := strings.TrimSpace(traceID)
	if !gcpTraceTraceIDRegex.MatchString(value) {
		return false
	}
	return strings.Trim(strings.ToLower(value), "0") != ""
}

func isGCPTraceValidSpanID(spanID string) bool {
	value := strings.TrimSpace(spanID)
	if !gcpTraceSpanIDRegex.MatchString(value) {
		return false
	}
	return strings.Trim(strings.ToLower(value), "0") != ""
}

func isGCPTraceMissingProject(project string) bool {
	project = strings.ToLower(strings.TrimSpace(project))
	return strings.Contains(project, "missing") || strings.Contains(project, "deleted")
}

func gcpTraceNormalizeSpanKind(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "SPAN_KIND_UNSPECIFIED"
	}
	value = strings.ToUpper(value)
	if strings.HasPrefix(value, "SPAN_KIND_") {
		return value
	}
	return "SPAN_KIND_" + value
}

func gcpTraceParseTimestamp(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func decodeGCPTraceJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPTraceInvalidArgument(w, path, "request body could not be read")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	body := map[string]any{}
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPTraceInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func gcpTraceRespondValidationError(w http.ResponseWriter, path, code, message string) {
	switch code {
	case "NotFound":
		respondGCPTraceNotFound(w, path, message)
	case "FailedPrecondition":
		respondGCPTraceFailedPrecondition(w, path, message)
	default:
		respondGCPTraceInvalidArgument(w, path, message)
	}
}

func respondGCPTraceInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPTraceFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPTraceNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func gcpTraceString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key].(string); ok {
			return value
		}
	}
	return ""
}

func gcpTraceBodyMap(m map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if value, ok := m[key].(map[string]any); ok {
			return value
		}
	}
	return map[string]any{}
}

func gcpTraceOptionalBool(m map[string]any, keys ...string) *bool {
	for _, key := range keys {
		value, ok := m[key]
		if !ok {
			continue
		}
		if boolValue, ok := value.(bool); ok {
			return &boolValue
		}
		return nil
	}
	return nil
}

func handleGCPContractProbe_trace(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "trace") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPTraceInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		span := gcpTraceNormalizedSpan{
			Name:        "projects/stackyard/traces/0123456789abcdef0123456789abcdef/spans/1111111111111111",
			Project:     "stackyard",
			TraceID:     "0123456789abcdef0123456789abcdef",
			SpanID:      "1111111111111111",
			DisplayName: "stackyard.trace.contract_probe",
			StartTime:   gcpTraceReferenceTime,
			EndTime:     gcpTraceReferenceTime.Add(250 * time.Millisecond),
			SpanKind:    "SPAN_KIND_SERVER",
			SameProcess: true,
		}
		response := gcpTraceSpanJSONFixture(span)
		response["service"] = "trace"
		response["provider"] = providerGCP
		response["path"] = path
		respondJSON(w, http.StatusOK, response)
		return true
	}
	return false
}
