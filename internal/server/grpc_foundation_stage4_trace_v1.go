package server

import (
	"strconv"
	"strings"
	"time"

	tracev1pb "cloud.google.com/go/trace/apiv1/tracepb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpTraceV1ListTracesMethod  = "/google.devtools.cloudtrace.v1.TraceService/ListTraces"
	gcpTraceV1GetTraceMethod    = "/google.devtools.cloudtrace.v1.TraceService/GetTrace"
	gcpTraceV1PatchTracesMethod = "/google.devtools.cloudtrace.v1.TraceService/PatchTraces"
)

func gcpStage4GRPCTraceV1(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpTraceV1ListTracesMethod:
		return gcpStage4GRPCTraceV1ListTraces(grpcReqBody)
	case gcpTraceV1GetTraceMethod:
		return gcpStage4GRPCTraceV1GetTrace(grpcReqBody)
	case gcpTraceV1PatchTracesMethod:
		return gcpStage4GRPCTraceV1PatchTraces(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCTraceV1ListTraces(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tracev1pb.ListTracesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}

	project := strings.TrimSpace(req.GetProjectId())
	if project == "" {
		return grpcInvalidArgument("project_id-required")
	}
	if !isGCPTraceV1ValidProjectID(project) {
		return grpcInvalidArgument("project_id-invalid")
	}
	if isGCPTraceV1MissingProject(project) {
		return grpcNotFound("project-not-found")
	}

	view, ok := gcpTraceV1ViewFromInt(int(req.GetView()))
	if !ok {
		return grpcInvalidArgument("view-invalid")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-negative")
	}
	if req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-too-large")
	}

	offset := 0
	if token := strings.TrimSpace(req.GetPageToken()); token != "" {
		parsed, err := strconv.Atoi(token)
		if err != nil || parsed < 0 {
			return grpcInvalidArgument("page_token-invalid")
		}
		offset = parsed
	}

	startTime, hasStart, ok := gcpTraceV1TimestampFromProto(req.GetStartTime())
	if !ok {
		return grpcInvalidArgument("start_time-invalid")
	}
	endTime, hasEnd, ok := gcpTraceV1TimestampFromProto(req.GetEndTime())
	if !ok {
		return grpcInvalidArgument("end_time-invalid")
	}
	if hasStart && hasEnd && endTime.Before(startTime) {
		return grpcInvalidArgument("time_range-invalid")
	}

	filterSpec, ok := gcpTraceV1ParseFilterSpec(req.GetFilter())
	if !ok {
		return grpcInvalidArgument("filter-unsupported")
	}
	orderKey, orderDesc, ok := gcpTraceV1ParseOrderBy(req.GetOrderBy())
	if !ok {
		return grpcInvalidArgument("order_by-invalid")
	}

	opts := gcpTraceV1ListOptions{
		View:      view,
		PageSize:  int(req.GetPageSize()),
		Offset:    offset,
		Filter:    filterSpec,
		OrderKey:  orderKey,
		OrderDesc: orderDesc,
	}
	if hasStart {
		value := startTime.UTC()
		opts.StartTime = &value
	}
	if hasEnd {
		value := endTime.UTC()
		opts.EndTime = &value
	}

	traces, nextPageToken, code, message := gcpTraceV1ListTraces(project, opts)
	if code != "" {
		return gcpTraceV1GRPCContractError(code, message)
	}
	out := make([]*tracev1pb.Trace, 0, len(traces))
	for _, trace := range traces {
		out = append(out, gcpTraceV1TraceToProto(trace, view))
	}

	return grpcProtoSuccess(&tracev1pb.ListTracesResponse{
		Traces:        out,
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCTraceV1GetTrace(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tracev1pb.GetTraceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	trace, code, message := gcpTraceV1ResolveTrace(strings.TrimSpace(req.GetProjectId()), strings.TrimSpace(req.GetTraceId()))
	if code != "" {
		return gcpTraceV1GRPCContractError(code, message)
	}
	return grpcProtoSuccess(gcpTraceV1TraceToProto(trace, gcpTraceV1ViewComplete))
}

func gcpStage4GRPCTraceV1PatchTraces(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tracev1pb.PatchTracesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project := strings.TrimSpace(req.GetProjectId())
	if project == "" {
		return grpcInvalidArgument("project_id-required")
	}
	if !isGCPTraceV1ValidProjectID(project) {
		return grpcInvalidArgument("project_id-invalid")
	}
	if isGCPTraceV1MissingProject(project) {
		return grpcNotFound("project-not-found")
	}
	code, reason := gcpTraceV1ValidatePatchProto(project, req.GetTraces())
	if code != "" {
		switch code {
		case "NotFound":
			return grpcNotFound(reason)
		case "FailedPrecondition":
			return grpcFailedPrecondition(reason)
		default:
			return grpcInvalidArgument(reason)
		}
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpTraceV1ValidatePatchProto(project string, traces *tracev1pb.Traces) (string, string) {
	if traces == nil || len(traces.GetTraces()) == 0 {
		return "InvalidArgument", "traces-required"
	}
	for i, trace := range traces.GetTraces() {
		if trace == nil {
			return "InvalidArgument", "traces[" + strconv.Itoa(i) + "]-required"
		}
		if traceProject := strings.TrimSpace(trace.GetProjectId()); traceProject != "" && traceProject != project {
			return "InvalidArgument", "trace-project-mismatch"
		}
		traceID := strings.TrimSpace(trace.GetTraceId())
		if traceID == "" {
			return "InvalidArgument", "trace_id-required"
		}
		if !isGCPTraceV1ValidTraceID(traceID) {
			return "InvalidArgument", "trace_id-invalid"
		}
		if len(trace.GetSpans()) == 0 {
			return "InvalidArgument", "spans-required"
		}
		for j, span := range trace.GetSpans() {
			if span == nil {
				return "InvalidArgument", "span[" + strconv.Itoa(j) + "]-required"
			}
			if span.GetSpanId() == 0 {
				return "InvalidArgument", "span_id-invalid"
			}
			if span.GetParentSpanId() != 0 && span.GetParentSpanId() == span.GetSpanId() {
				return "InvalidArgument", "parent_span_id-self"
			}
			if int(span.GetKind()) < 0 || int(span.GetKind()) > 2 {
				return "InvalidArgument", "span_kind-invalid"
			}
			if strings.TrimSpace(span.GetName()) == "" {
				return "InvalidArgument", "span_name-required"
			}
			if len([]byte(span.GetName())) > 128 {
				return "InvalidArgument", "span_name-too-long"
			}

			startTime, hasStart, ok := gcpTraceV1TimestampFromProto(span.GetStartTime())
			if !ok || !hasStart {
				return "InvalidArgument", "start_time-required"
			}
			endTime, hasEnd, ok := gcpTraceV1TimestampFromProto(span.GetEndTime())
			if !ok || !hasEnd {
				return "InvalidArgument", "end_time-required"
			}
			if endTime.Before(startTime) {
				return "InvalidArgument", "time_range-invalid"
			}
			if endTime.Sub(startTime) > 24*time.Hour {
				return "FailedPrecondition", "span-duration-exceeds-staged-limit"
			}
			for key := range span.GetLabels() {
				if strings.TrimSpace(key) == "" {
					return "InvalidArgument", "label-key-empty"
				}
			}
		}
	}
	return "", ""
}

func gcpTraceV1TraceToProto(trace gcpTraceV1Trace, view int) *tracev1pb.Trace {
	out := &tracev1pb.Trace{
		ProjectId: trace.ProjectID,
		TraceId:   trace.TraceID,
	}
	for _, span := range gcpTraceV1TraceSpansByView(trace, view) {
		item := &tracev1pb.TraceSpan{
			SpanId:    span.SpanID,
			Kind:      tracev1pb.TraceSpan_SpanKind(span.Kind),
			Name:      span.Name,
			StartTime: timestamppb.New(span.StartTime),
			EndTime:   timestamppb.New(span.EndTime),
			Labels:    span.Labels,
		}
		if span.ParentSpanID != 0 {
			item.ParentSpanId = span.ParentSpanID
		}
		out.Spans = append(out.Spans, item)
	}
	return out
}

func gcpTraceV1TimestampFromProto(ts *timestamppb.Timestamp) (time.Time, bool, bool) {
	if ts == nil {
		return time.Time{}, false, true
	}
	if err := ts.CheckValid(); err != nil {
		return time.Time{}, false, false
	}
	return ts.AsTime(), true, true
}

func gcpTraceV1GRPCContractError(code, message string) ([]byte, string, string, bool) {
	reason := gcpTraceV1GRPCReasonForMessage(message)
	switch code {
	case "NotFound":
		return grpcNotFound(reason)
	case "FailedPrecondition":
		return grpcFailedPrecondition(reason)
	default:
		return grpcInvalidArgument(reason)
	}
}

func gcpTraceV1GRPCReasonForMessage(message string) string {
	switch {
	case strings.Contains(message, "project_id is required"):
		return "project_id-required"
	case strings.Contains(message, "project_id is invalid"):
		return "project_id-invalid"
	case strings.Contains(message, "trace_id is required"):
		return "trace_id-required"
	case strings.Contains(message, "trace_id is invalid"):
		return "trace_id-invalid"
	case strings.Contains(message, "trace not found"):
		return "trace-not-found"
	case strings.Contains(message, "pageToken is out of range"):
		return "page_token-out-of-range"
	default:
		return "invalid-argument"
	}
}
