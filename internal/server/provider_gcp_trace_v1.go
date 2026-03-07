package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	gcpTraceV1GRPCListTracesPath  = "/gcp/google.devtools.cloudtrace.v1.TraceService/ListTraces"
	gcpTraceV1GRPCGetTracePath    = "/gcp/google.devtools.cloudtrace.v1.TraceService/GetTrace"
	gcpTraceV1GRPCPatchTracesPath = "/gcp/google.devtools.cloudtrace.v1.TraceService/PatchTraces"

	gcpTraceV1ViewMinimal  = 1
	gcpTraceV1ViewRootSpan = 2
	gcpTraceV1ViewComplete = 3
)

var (
	gcpTraceV1ReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpTraceV1ProjectRegex  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,62}$`)
	gcpTraceV1TraceIDRegex  = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)
)

type gcpTraceV1Trace struct {
	ProjectID string
	TraceID   string
	Spans     []gcpTraceV1Span
}

type gcpTraceV1Span struct {
	SpanID       uint64
	Kind         int
	Name         string
	StartTime    time.Time
	EndTime      time.Time
	ParentSpanID uint64
	Labels       map[string]string
}

type gcpTraceV1FilterSpec struct {
	kind    string
	key     string
	value   string
	isExact bool
}

type gcpTraceV1ListOptions struct {
	View       int
	PageSize   int
	Offset     int
	StartTime  *time.Time
	EndTime    *time.Time
	Filter     gcpTraceV1FilterSpec
	OrderKey   string
	OrderDesc  bool
	RawOrderBy string
}

func (s *Server) handleGCPTraceV1Router(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_trace_v1(w, r) {
		return true
	}

	path := normalizeGCPTraceV1Path(rawRequestPath(r))
	if !isGCPTraceV1Path(path, hasGCPTraceV1Hint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPTraceV1ListTracesGET(w, r, path) {
			return true
		}
		if handleGCPTraceV1GetTraceGET(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		body, valid := decodeGCPTraceV1JSONBody(w, r, path)
		if !valid {
			return true
		}
		if handleGCPTraceV1PatchTracesREST(w, path, body) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		body, valid := decodeGCPTraceV1JSONBody(w, r, path)
		if !valid {
			return true
		}
		if handleGCPTraceV1ListTracesGRPCJSON(w, path, body) {
			return true
		}
		if handleGCPTraceV1GetTraceGRPCJSON(w, path, body) {
			return true
		}
		if handleGCPTraceV1PatchTracesGRPCJSON(w, path, body) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPTraceV1Path(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPTraceV1Hint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "trace_v1",
		"trace-v1",
		"trace-apiv1",
		"trace_apiv1",
		"stackdriver-trace-v1",
		"stackdriver_trace_v1",
		"cloudtrace-v1",
		"gcp-trace-v1":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-trace-apiv1") || strings.Contains(ua, "cloud.google.com/go/trace/apiv1")
}

func isGCPTraceV1Path(path string, includeHint bool) bool {
	if path == gcpTraceV1GRPCListTracesPath || path == gcpTraceV1GRPCGetTracePath || path == gcpTraceV1GRPCPatchTracesPath {
		return true
	}
	if _, ok := parseGCPTraceV1TracesCollectionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPTraceV1TracePath(path); ok {
		return true
	}
	return includeHint && strings.HasPrefix(path, "/gcp/v1/projects/") && strings.Contains(path, "/traces")
}

func handleGCPTraceV1ListTracesGET(w http.ResponseWriter, r *http.Request, path string) bool {
	project, ok := parseGCPTraceV1TracesCollectionPath(path)
	if !ok {
		return false
	}
	listOpts, code, message := gcpTraceV1ListOptionsFromQuery(project, r, path)
	if code != "" {
		gcpTraceV1RespondError(w, path, code, message)
		return true
	}
	if isGCPTraceV1MissingProject(project) {
		respondGCPTraceV1NotFound(w, path, "project not found")
		return true
	}

	response, code, message := gcpTraceV1BuildListResponse(project, listOpts)
	if code != "" {
		gcpTraceV1RespondError(w, path, code, message)
		return true
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPTraceV1GetTraceGET(w http.ResponseWriter, path string) bool {
	project, traceID, ok := parseGCPTraceV1TracePath(path)
	if !ok {
		return false
	}
	trace, code, message := gcpTraceV1ResolveTrace(project, traceID)
	if code != "" {
		gcpTraceV1RespondError(w, path, code, message)
		return true
	}
	respondJSON(w, http.StatusOK, gcpTraceV1TraceToJSON(trace, gcpTraceV1ViewComplete))
	return true
}

func handleGCPTraceV1PatchTracesREST(w http.ResponseWriter, path string, body map[string]any) bool {
	project, ok := parseGCPTraceV1TracesCollectionPath(path)
	if !ok {
		return false
	}
	code, message := gcpTraceV1ValidatePatchBody(project, body, false)
	if code != "" {
		gcpTraceV1RespondError(w, path, code, message)
		return true
	}
	if isGCPTraceV1MissingProject(project) {
		respondGCPTraceV1NotFound(w, path, "project not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTraceV1ListTracesGRPCJSON(w http.ResponseWriter, path string, body map[string]any) bool {
	if path != gcpTraceV1GRPCListTracesPath {
		return false
	}
	project := strings.TrimSpace(gcpTraceV1String(body, "projectId", "project_id"))
	listOpts, code, message := gcpTraceV1ListOptionsFromBody(project, body)
	if code != "" {
		gcpTraceV1RespondError(w, path, code, message)
		return true
	}
	if isGCPTraceV1MissingProject(project) {
		respondGCPTraceV1NotFound(w, path, "project not found")
		return true
	}

	response, code, message := gcpTraceV1BuildListResponse(project, listOpts)
	if code != "" {
		gcpTraceV1RespondError(w, path, code, message)
		return true
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPTraceV1GetTraceGRPCJSON(w http.ResponseWriter, path string, body map[string]any) bool {
	if path != gcpTraceV1GRPCGetTracePath {
		return false
	}
	project := strings.TrimSpace(gcpTraceV1String(body, "projectId", "project_id"))
	traceID := strings.TrimSpace(gcpTraceV1String(body, "traceId", "trace_id"))
	trace, code, message := gcpTraceV1ResolveTrace(project, traceID)
	if code != "" {
		gcpTraceV1RespondError(w, path, code, message)
		return true
	}
	respondJSON(w, http.StatusOK, gcpTraceV1TraceToJSON(trace, gcpTraceV1ViewComplete))
	return true
}

func handleGCPTraceV1PatchTracesGRPCJSON(w http.ResponseWriter, path string, body map[string]any) bool {
	if path != gcpTraceV1GRPCPatchTracesPath {
		return false
	}
	project := strings.TrimSpace(gcpTraceV1String(body, "projectId", "project_id"))
	code, message := gcpTraceV1ValidatePatchBody(project, body, true)
	if code != "" {
		gcpTraceV1RespondError(w, path, code, message)
		return true
	}
	if isGCPTraceV1MissingProject(project) {
		respondGCPTraceV1NotFound(w, path, "project not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func gcpTraceV1ResolveTrace(project, traceID string) (gcpTraceV1Trace, string, string) {
	project = strings.TrimSpace(project)
	traceID = strings.TrimSpace(traceID)
	if project == "" {
		return gcpTraceV1Trace{}, "InvalidArgument", "project_id is required"
	}
	if !isGCPTraceV1ValidProjectID(project) {
		return gcpTraceV1Trace{}, "InvalidArgument", "project_id is invalid"
	}
	if isGCPTraceV1MissingProject(project) {
		return gcpTraceV1Trace{}, "NotFound", "project not found"
	}
	if traceID == "" {
		return gcpTraceV1Trace{}, "InvalidArgument", "trace_id is required"
	}
	if !isGCPTraceV1ValidTraceID(traceID) {
		return gcpTraceV1Trace{}, "InvalidArgument", "trace_id is invalid"
	}

	traceID = strings.ToLower(traceID)
	for _, trace := range gcpTraceV1SeedTraces(project) {
		if trace.TraceID == traceID {
			return trace, "", ""
		}
	}
	return gcpTraceV1Trace{}, "NotFound", "trace not found"
}

func gcpTraceV1BuildListResponse(project string, listOpts gcpTraceV1ListOptions) (map[string]any, string, string) {
	traces, nextPageToken, code, message := gcpTraceV1ListTraces(project, listOpts)
	if code != "" {
		return nil, code, message
	}
	out := make([]any, 0, len(traces))
	for _, trace := range traces {
		out = append(out, gcpTraceV1TraceToJSON(trace, listOpts.View))
	}
	return map[string]any{
		"traces":        out,
		"nextPageToken": nextPageToken,
	}, "", ""
}

func gcpTraceV1ListTraces(project string, listOpts gcpTraceV1ListOptions) ([]gcpTraceV1Trace, string, string, string) {
	if !isGCPTraceV1ValidProjectID(project) {
		return nil, "", "InvalidArgument", "project_id is invalid"
	}
	if isGCPTraceV1MissingProject(project) {
		return nil, "", "NotFound", "project not found"
	}

	traces := append([]gcpTraceV1Trace(nil), gcpTraceV1SeedTraces(project)...)
	filtered := make([]gcpTraceV1Trace, 0, len(traces))
	for _, trace := range traces {
		if gcpTraceV1TraceMatchesTimeWindow(trace, listOpts.StartTime, listOpts.EndTime) &&
			gcpTraceV1TraceMatchesFilter(trace, listOpts.Filter) {
			filtered = append(filtered, trace)
		}
	}

	gcpTraceV1SortTraces(filtered, listOpts.OrderKey, listOpts.OrderDesc)
	if listOpts.Offset > len(filtered) {
		return nil, "", "InvalidArgument", "pageToken is out of range"
	}

	start := listOpts.Offset
	end := len(filtered)
	if listOpts.PageSize > 0 && start+listOpts.PageSize < end {
		end = start + listOpts.PageSize
	}
	nextPageToken := ""
	if end < len(filtered) {
		nextPageToken = strconv.Itoa(end)
	}
	return filtered[start:end], nextPageToken, "", ""
}

func gcpTraceV1ListOptionsFromQuery(project string, r *http.Request, path string) (gcpTraceV1ListOptions, string, string) {
	if strings.TrimSpace(project) == "" {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "project_id is required"
	}
	if !isGCPTraceV1ValidProjectID(project) {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "project_id is invalid"
	}
	query := r.URL.Query()
	return gcpTraceV1ListOptionsFromRaw(
		query.Get("view"),
		gcpTraceV1FirstNonEmpty(query.Get("pageSize"), query.Get("page_size")),
		gcpTraceV1FirstNonEmpty(query.Get("pageToken"), query.Get("page_token")),
		gcpTraceV1FirstNonEmpty(query.Get("startTime"), query.Get("start_time")),
		gcpTraceV1FirstNonEmpty(query.Get("endTime"), query.Get("end_time")),
		query.Get("filter"),
		gcpTraceV1FirstNonEmpty(query.Get("orderBy"), query.Get("order_by")),
		path,
	)
}

func gcpTraceV1ListOptionsFromBody(project string, body map[string]any) (gcpTraceV1ListOptions, string, string) {
	if strings.TrimSpace(project) == "" {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "project_id is required"
	}
	if !isGCPTraceV1ValidProjectID(project) {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "project_id is invalid"
	}
	return gcpTraceV1ListOptionsFromRaw(
		gcpTraceV1Any(body, "view"),
		gcpTraceV1Any(body, "pageSize", "page_size"),
		gcpTraceV1Any(body, "pageToken", "page_token"),
		gcpTraceV1String(body, "startTime", "start_time"),
		gcpTraceV1String(body, "endTime", "end_time"),
		gcpTraceV1String(body, "filter"),
		gcpTraceV1String(body, "orderBy", "order_by"),
		"",
	)
}

func gcpTraceV1ListOptionsFromRaw(viewRaw, pageSizeRaw, pageTokenRaw any, startRaw, endRaw, filterRaw, orderRaw, path string) (gcpTraceV1ListOptions, string, string) {
	view, ok := gcpTraceV1ParseView(viewRaw)
	if !ok {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "view is invalid"
	}

	pageSize, ok := gcpTraceV1ParseInt(pageSizeRaw)
	if !ok {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "pageSize must be an integer"
	}
	if pageSize < 0 {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "pageSize must be non-negative"
	}
	if pageSize > 1000 {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "pageSize must be <= 1000"
	}

	offset, ok := gcpTraceV1ParseInt(pageTokenRaw)
	if !ok {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "pageToken is invalid"
	}
	if offset < 0 {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "pageToken is invalid"
	}

	startTime, hasStart, ok := gcpTraceV1ParseTimestamp(startRaw)
	if !ok {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "startTime must be RFC3339"
	}
	endTime, hasEnd, ok := gcpTraceV1ParseTimestamp(endRaw)
	if !ok {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "endTime must be RFC3339"
	}
	if hasStart && hasEnd && endTime.Before(startTime) {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "endTime must be >= startTime"
	}

	filterSpec, ok := gcpTraceV1ParseFilterSpec(filterRaw)
	if !ok {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "filter is unsupported in staged emulation"
	}
	orderKey, orderDesc, ok := gcpTraceV1ParseOrderBy(orderRaw)
	if !ok {
		return gcpTraceV1ListOptions{}, "InvalidArgument", "orderBy is invalid"
	}

	opts := gcpTraceV1ListOptions{
		View:       view,
		PageSize:   pageSize,
		Offset:     offset,
		Filter:     filterSpec,
		OrderKey:   orderKey,
		OrderDesc:  orderDesc,
		RawOrderBy: strings.TrimSpace(orderRaw),
	}
	if hasStart {
		value := startTime.UTC()
		opts.StartTime = &value
	}
	if hasEnd {
		value := endTime.UTC()
		opts.EndTime = &value
	}
	_ = path
	return opts, "", ""
}

func gcpTraceV1ValidatePatchBody(project string, body map[string]any, allowProjectInBody bool) (string, string) {
	project = strings.TrimSpace(project)
	if project == "" {
		return "InvalidArgument", "project_id is required"
	}
	if !isGCPTraceV1ValidProjectID(project) {
		return "InvalidArgument", "project_id is invalid"
	}

	var tracesAny []any
	if allowProjectInBody {
		bodyProject := strings.TrimSpace(gcpTraceV1String(body, "projectId", "project_id"))
		if bodyProject != "" && bodyProject != project {
			return "InvalidArgument", "project_id must match request path"
		}
		if tracesWrapper, ok := body["traces"].(map[string]any); ok {
			tracesAny, _ = tracesWrapper["traces"].([]any)
		}
		if tracesAny == nil {
			tracesAny, _ = body["traces"].([]any)
		}
	} else {
		tracesAny, _ = body["traces"].([]any)
		if tracesAny == nil {
			if tracesWrapper, ok := body["traces"].(map[string]any); ok {
				tracesAny, _ = tracesWrapper["traces"].([]any)
			}
		}
	}

	if len(tracesAny) == 0 {
		return "InvalidArgument", "traces is required"
	}

	for i, rawTrace := range tracesAny {
		traceMap, ok := rawTrace.(map[string]any)
		if !ok {
			return "InvalidArgument", fmt.Sprintf("traces[%d] must be an object", i)
		}
		if traceProject := strings.TrimSpace(gcpTraceV1String(traceMap, "projectId", "project_id")); traceProject != "" && traceProject != project {
			return "InvalidArgument", fmt.Sprintf("traces[%d].projectId must match request project", i)
		}
		traceID := strings.TrimSpace(gcpTraceV1String(traceMap, "traceId", "trace_id"))
		if traceID == "" {
			return "InvalidArgument", fmt.Sprintf("traces[%d].traceId is required", i)
		}
		if !isGCPTraceV1ValidTraceID(traceID) {
			return "InvalidArgument", fmt.Sprintf("traces[%d].traceId is invalid", i)
		}

		spansAny, _ := traceMap["spans"].([]any)
		if len(spansAny) == 0 {
			return "InvalidArgument", fmt.Sprintf("traces[%d].spans must include at least one span", i)
		}
		for spanIndex, rawSpan := range spansAny {
			spanMap, ok := rawSpan.(map[string]any)
			if !ok {
				return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d] must be an object", i, spanIndex)
			}

			spanID, ok := gcpTraceV1ParseUint64(gcpTraceV1Any(spanMap, "spanId", "span_id"))
			if !ok || spanID == 0 {
				return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d].spanId is invalid", i, spanIndex)
			}
			if parentRaw := gcpTraceV1Any(spanMap, "parentSpanId", "parent_span_id"); parentRaw != nil {
				parentSpanID, ok := gcpTraceV1ParseUint64(parentRaw)
				if !ok {
					return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d].parentSpanId is invalid", i, spanIndex)
				}
				if parentSpanID == spanID {
					return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d].parentSpanId cannot equal spanId", i, spanIndex)
				}
			}

			kind, ok := gcpTraceV1ParseSpanKind(gcpTraceV1Any(spanMap, "kind"))
			if !ok {
				return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d].kind is invalid", i, spanIndex)
			}
			_ = kind

			spanName := strings.TrimSpace(gcpTraceV1String(spanMap, "name"))
			if spanName == "" {
				return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d].name is required", i, spanIndex)
			}
			if len([]byte(spanName)) > 128 {
				return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d].name must be <= 128 bytes", i, spanIndex)
			}

			startTime, hasStart, ok := gcpTraceV1ParseTimestamp(gcpTraceV1String(spanMap, "startTime", "start_time"))
			if !ok || !hasStart {
				return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d].startTime is required", i, spanIndex)
			}
			endTime, hasEnd, ok := gcpTraceV1ParseTimestamp(gcpTraceV1String(spanMap, "endTime", "end_time"))
			if !ok || !hasEnd {
				return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d].endTime is required", i, spanIndex)
			}
			if endTime.Before(startTime) {
				return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d].endTime must be >= startTime", i, spanIndex)
			}
			if endTime.Sub(startTime) > 24*time.Hour {
				return "FailedPrecondition", fmt.Sprintf("traces[%d].spans[%d] duration exceeds staged limit", i, spanIndex)
			}

			if labelsMap, ok := spanMap["labels"].(map[string]any); ok {
				for labelKey, labelValue := range labelsMap {
					if strings.TrimSpace(labelKey) == "" {
						return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d].labels contains empty key", i, spanIndex)
					}
					if _, ok := labelValue.(string); !ok {
						return "InvalidArgument", fmt.Sprintf("traces[%d].spans[%d].labels[%q] must be a string", i, spanIndex, labelKey)
					}
				}
			}
		}
	}

	return "", ""
}

func gcpTraceV1SeedTraces(project string) []gcpTraceV1Trace {
	traceA := "0123456789abcdef0123456789abcdef"
	traceB := "fedcba9876543210fedcba9876543210"

	return []gcpTraceV1Trace{
		{
			ProjectID: project,
			TraceID:   traceA,
			Spans: []gcpTraceV1Span{
				{
					SpanID:    1001,
					Kind:      1,
					Name:      "/stackyard/tracev1/root",
					StartTime: gcpTraceV1ReferenceTime,
					EndTime:   gcpTraceV1ReferenceTime.Add(750 * time.Millisecond),
					Labels: map[string]string{
						"/component": "trace_v1",
						"/http/path": "/stackyard/tracev1/root",
					},
				},
				{
					SpanID:       1002,
					ParentSpanID: 1001,
					Kind:         2,
					Name:         "/stackyard/tracev1/downstream",
					StartTime:    gcpTraceV1ReferenceTime.Add(100 * time.Millisecond),
					EndTime:      gcpTraceV1ReferenceTime.Add(500 * time.Millisecond),
					Labels: map[string]string{
						"/component": "trace_v1_child",
					},
				},
			},
		},
		{
			ProjectID: project,
			TraceID:   traceB,
			Spans: []gcpTraceV1Span{
				{
					SpanID:    2001,
					Kind:      1,
					Name:      "/stackyard/tracev1/secondary",
					StartTime: gcpTraceV1ReferenceTime.Add(2 * time.Second),
					EndTime:   gcpTraceV1ReferenceTime.Add(3 * time.Second),
					Labels: map[string]string{
						"/component": "trace_v1",
					},
				},
			},
		},
	}
}

func gcpTraceV1TraceToJSON(trace gcpTraceV1Trace, view int) map[string]any {
	out := map[string]any{
		"projectId": trace.ProjectID,
		"traceId":   trace.TraceID,
	}

	spans := gcpTraceV1TraceSpansByView(trace, view)
	if len(spans) == 0 {
		return out
	}

	spanOut := make([]any, 0, len(spans))
	for _, span := range spans {
		item := map[string]any{
			"spanId":    strconv.FormatUint(span.SpanID, 10),
			"kind":      span.Kind,
			"name":      span.Name,
			"startTime": span.StartTime.UTC().Format(time.RFC3339Nano),
			"endTime":   span.EndTime.UTC().Format(time.RFC3339Nano),
			"labels":    span.Labels,
		}
		if span.ParentSpanID != 0 {
			item["parentSpanId"] = strconv.FormatUint(span.ParentSpanID, 10)
		}
		spanOut = append(spanOut, item)
	}
	out["spans"] = spanOut
	return out
}

func gcpTraceV1TraceSpansByView(trace gcpTraceV1Trace, view int) []gcpTraceV1Span {
	switch view {
	case gcpTraceV1ViewMinimal:
		return nil
	case gcpTraceV1ViewRootSpan:
		for _, span := range trace.Spans {
			if span.ParentSpanID == 0 {
				return []gcpTraceV1Span{span}
			}
		}
		return nil
	default:
		return append([]gcpTraceV1Span(nil), trace.Spans...)
	}
}

func gcpTraceV1TraceMatchesTimeWindow(trace gcpTraceV1Trace, start, end *time.Time) bool {
	if start == nil && end == nil {
		return true
	}
	traceStart, traceEnd := gcpTraceV1TraceBounds(trace)
	if start != nil && traceEnd.Before(*start) {
		return false
	}
	if end != nil && traceStart.After(*end) {
		return false
	}
	return true
}

func gcpTraceV1TraceMatchesFilter(trace gcpTraceV1Trace, spec gcpTraceV1FilterSpec) bool {
	if spec.kind == "" {
		return true
	}
	switch spec.kind {
	case "root":
		for _, span := range trace.Spans {
			if span.ParentSpanID != 0 {
				continue
			}
			if gcpTraceV1StringMatch(span.Name, spec.value, spec.isExact) {
				return true
			}
		}
		return false
	case "span":
		for _, span := range trace.Spans {
			if gcpTraceV1StringMatch(span.Name, spec.value, spec.isExact) {
				return true
			}
		}
		return false
	case "label-key":
		for _, span := range trace.Spans {
			if _, ok := span.Labels[spec.key]; ok {
				return true
			}
		}
		return false
	case "label-value":
		for _, span := range trace.Spans {
			value, ok := span.Labels[spec.key]
			if !ok {
				continue
			}
			if gcpTraceV1StringMatch(value, spec.value, spec.isExact) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func gcpTraceV1SortTraces(traces []gcpTraceV1Trace, key string, desc bool) {
	sort.SliceStable(traces, func(i, j int) bool {
		left := traces[i]
		right := traces[j]
		less := false
		switch key {
		case "name":
			less = gcpTraceV1RootSpanName(left) < gcpTraceV1RootSpanName(right)
		case "duration":
			less = gcpTraceV1TraceDuration(left) < gcpTraceV1TraceDuration(right)
		case "start":
			less = gcpTraceV1TraceStart(left).Before(gcpTraceV1TraceStart(right))
		default:
			less = left.TraceID < right.TraceID
		}
		if desc {
			return !less
		}
		return less
	})
}

func gcpTraceV1TraceDuration(trace gcpTraceV1Trace) time.Duration {
	start, end := gcpTraceV1TraceBounds(trace)
	return end.Sub(start)
}

func gcpTraceV1TraceStart(trace gcpTraceV1Trace) time.Time {
	start, _ := gcpTraceV1TraceBounds(trace)
	return start
}

func gcpTraceV1TraceBounds(trace gcpTraceV1Trace) (time.Time, time.Time) {
	if len(trace.Spans) == 0 {
		return gcpTraceV1ReferenceTime, gcpTraceV1ReferenceTime
	}
	start := trace.Spans[0].StartTime
	end := trace.Spans[0].EndTime
	for _, span := range trace.Spans[1:] {
		if span.StartTime.Before(start) {
			start = span.StartTime
		}
		if span.EndTime.After(end) {
			end = span.EndTime
		}
	}
	return start, end
}

func gcpTraceV1RootSpanName(trace gcpTraceV1Trace) string {
	for _, span := range trace.Spans {
		if span.ParentSpanID == 0 {
			return span.Name
		}
	}
	if len(trace.Spans) == 0 {
		return ""
	}
	return trace.Spans[0].Name
}

func gcpTraceV1StringMatch(candidate, query string, exact bool) bool {
	if exact {
		return candidate == query
	}
	return strings.HasPrefix(candidate, query)
}

func gcpTraceV1ParseFilterSpec(raw string) (gcpTraceV1FilterSpec, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return gcpTraceV1FilterSpec{}, true
	}
	if strings.Contains(value, " ") {
		return gcpTraceV1FilterSpec{}, false
	}
	if strings.HasPrefix(value, "+root:") {
		query := strings.TrimPrefix(value, "+root:")
		return gcpTraceV1FilterSpec{kind: "root", value: query, isExact: true}, query != ""
	}
	if strings.HasPrefix(value, "root:") {
		query := strings.TrimPrefix(value, "root:")
		return gcpTraceV1FilterSpec{kind: "root", value: query}, query != ""
	}
	if strings.HasPrefix(value, "+span:") {
		query := strings.TrimPrefix(value, "+span:")
		return gcpTraceV1FilterSpec{kind: "span", value: query, isExact: true}, query != ""
	}
	if strings.HasPrefix(value, "span:") {
		query := strings.TrimPrefix(value, "span:")
		return gcpTraceV1FilterSpec{kind: "span", value: query}, query != ""
	}
	if strings.HasPrefix(value, "label:") {
		key := strings.TrimPrefix(value, "label:")
		return gcpTraceV1FilterSpec{kind: "label-key", key: key}, key != ""
	}
	if strings.HasPrefix(value, "+") && strings.Contains(strings.TrimPrefix(value, "+"), ":") {
		key, query, ok := strings.Cut(strings.TrimPrefix(value, "+"), ":")
		key = strings.TrimSpace(key)
		query = strings.TrimSpace(query)
		return gcpTraceV1FilterSpec{kind: "label-value", key: key, value: query, isExact: true}, ok && key != "" && query != ""
	}
	if strings.Contains(value, ":") {
		key, query, ok := strings.Cut(value, ":")
		key = strings.TrimSpace(key)
		query = strings.TrimSpace(query)
		return gcpTraceV1FilterSpec{kind: "label-value", key: key, value: query}, ok && key != "" && query != ""
	}
	return gcpTraceV1FilterSpec{}, false
}

func gcpTraceV1ParseOrderBy(raw string) (key string, desc bool, ok bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "trace_id", false, true
	}
	if strings.HasSuffix(value, " desc") {
		desc = true
		value = strings.TrimSpace(strings.TrimSuffix(value, " desc"))
	}
	switch value {
	case "trace_id", "name", "duration", "start":
		return value, desc, true
	default:
		return "", false, false
	}
}

func gcpTraceV1ParseView(raw any) (int, bool) {
	switch value := raw.(type) {
	case nil:
		return gcpTraceV1ViewMinimal, true
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return gcpTraceV1ViewMinimal, true
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil {
			return gcpTraceV1ViewFromInt(parsed)
		}
		upper := strings.ToUpper(trimmed)
		upper = strings.TrimPrefix(upper, "VIEW_TYPE_")
		switch upper {
		case "UNSPECIFIED", "MINIMAL":
			return gcpTraceV1ViewMinimal, true
		case "ROOTSPAN":
			return gcpTraceV1ViewRootSpan, true
		case "COMPLETE":
			return gcpTraceV1ViewComplete, true
		default:
			return 0, false
		}
	case float64:
		if value != float64(int(value)) {
			return 0, false
		}
		return gcpTraceV1ViewFromInt(int(value))
	default:
		return 0, false
	}
}

func gcpTraceV1ViewFromInt(value int) (int, bool) {
	switch value {
	case 0, 1:
		return gcpTraceV1ViewMinimal, true
	case 2:
		return gcpTraceV1ViewRootSpan, true
	case 3:
		return gcpTraceV1ViewComplete, true
	default:
		return 0, false
	}
}

func gcpTraceV1ParseSpanKind(raw any) (int, bool) {
	switch value := raw.(type) {
	case nil:
		return 0, true
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return 0, true
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil {
			return gcpTraceV1ParseSpanKind(float64(parsed))
		}
		upper := strings.ToUpper(trimmed)
		upper = strings.TrimPrefix(upper, "SPAN_KIND_")
		switch upper {
		case "UNSPECIFIED":
			return 0, true
		case "RPC_SERVER":
			return 1, true
		case "RPC_CLIENT":
			return 2, true
		default:
			return 0, false
		}
	case float64:
		if value != float64(int(value)) {
			return 0, false
		}
		parsed := int(value)
		if parsed < 0 || parsed > 2 {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func gcpTraceV1ParseInt(raw any) (int, bool) {
	switch value := raw.(type) {
	case nil:
		return 0, true
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return 0, true
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	case float64:
		if value != float64(int(value)) {
			return 0, false
		}
		return int(value), true
	default:
		return 0, false
	}
}

func gcpTraceV1ParseUint64(raw any) (uint64, bool) {
	switch value := raw.(type) {
	case nil:
		return 0, false
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.ParseUint(trimmed, 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	case float64:
		if value != float64(uint64(value)) {
			return 0, false
		}
		if value < 0 {
			return 0, false
		}
		return uint64(value), true
	default:
		return 0, false
	}
}

func gcpTraceV1ParseTimestamp(raw string) (time.Time, bool, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false, true
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false, false
	}
	return parsed, true, true
}

func parseGCPTraceV1TracesCollectionPath(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "traces" {
		return "", false
	}
	project := strings.TrimSpace(parts[3])
	if project == "" {
		return "", false
	}
	return project, true
}

func parseGCPTraceV1TracePath(path string) (project, traceID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "traces" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	traceID = strings.TrimSpace(parts[5])
	if project == "" || traceID == "" {
		return "", "", false
	}
	return project, traceID, true
}

func isGCPTraceV1ValidProjectID(project string) bool {
	return gcpTraceV1ProjectRegex.MatchString(strings.TrimSpace(project))
}

func isGCPTraceV1ValidTraceID(traceID string) bool {
	value := strings.TrimSpace(traceID)
	if !gcpTraceV1TraceIDRegex.MatchString(value) {
		return false
	}
	return strings.Trim(strings.ToLower(value), "0") != ""
}

func isGCPTraceV1MissingProject(project string) bool {
	value := strings.ToLower(strings.TrimSpace(project))
	return strings.Contains(value, "missing") || strings.Contains(value, "deleted")
}

func respondGCPTraceV1InvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPTraceV1FailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPTraceV1NotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func gcpTraceV1RespondError(w http.ResponseWriter, path, code, message string) {
	switch code {
	case "NotFound":
		respondGCPTraceV1NotFound(w, path, message)
	case "FailedPrecondition":
		respondGCPTraceV1FailedPrecondition(w, path, message)
	default:
		respondGCPTraceV1InvalidArgument(w, path, message)
	}
}

func decodeGCPTraceV1JSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPTraceV1InvalidArgument(w, path, "request body could not be read")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	body := map[string]any{}
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPTraceV1InvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func gcpTraceV1String(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key].(string); ok {
			return value
		}
	}
	return ""
}

func gcpTraceV1Any(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value
		}
	}
	return nil
}

func gcpTraceV1FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func handleGCPContractProbe_trace_v1(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "trace_v1") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPTraceV1InvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/traces/0123456789abcdef0123456789abcdef",
			"projectId": "stackyard",
			"traceId":   "0123456789abcdef0123456789abcdef",
			"service":   "trace_v1",
			"provider":  providerGCP,
			"path":      path,
		})
		return true
	}
	return false
}
