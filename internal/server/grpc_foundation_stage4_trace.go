package server

import (
	"strconv"
	"strings"

	tracepb "cloud.google.com/go/trace/apiv2/tracepb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	gcpTraceBatchWriteSpansMethod = "/google.devtools.cloudtrace.v2.TraceService/BatchWriteSpans"
	gcpTraceCreateSpanMethod      = "/google.devtools.cloudtrace.v2.TraceService/CreateSpan"
)

func gcpStage4GRPCTrace(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpTraceBatchWriteSpansMethod:
		return gcpStage4GRPCTraceBatchWriteSpans(grpcReqBody)
	case gcpTraceCreateSpanMethod:
		return gcpStage4GRPCTraceCreateSpan(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCTraceBatchWriteSpans(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tracepb.BatchWriteSpansRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}

	project, ok := parseGCPTraceProjectName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if !isGCPTraceValidProjectID(project) {
		return grpcInvalidArgument("project-invalid")
	}
	if isGCPTraceMissingProject(project) {
		return grpcNotFound("project-not-found")
	}
	if len(req.GetSpans()) == 0 {
		return grpcInvalidArgument("spans-required")
	}

	for i, span := range req.GetSpans() {
		input, ok := gcpTraceSpanInputFromProto(span)
		if !ok {
			return grpcInvalidArgument("spans[" + strconv.Itoa(i) + "]-required")
		}
		_, code, message := gcpTraceValidateAndNormalizeSpanInput(input, project, "", "")
		if code != "" {
			return gcpTraceGRPCValidationError(code, message)
		}
	}

	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCTraceCreateSpan(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tracepb.Span{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}

	input, ok := gcpTraceSpanInputFromProto(req)
	if !ok {
		return grpcInvalidArgument("span-required")
	}
	normalized, code, message := gcpTraceValidateAndNormalizeSpanInput(input, "", "", "")
	if code != "" {
		return gcpTraceGRPCValidationError(code, message)
	}
	if isGCPTraceMissingProject(normalized.Project) {
		return grpcNotFound("project-not-found")
	}

	return grpcProtoSuccess(gcpTraceSpanProtoFixture(normalized))
}

func gcpTraceSpanInputFromProto(span *tracepb.Span) (gcpTraceSpanInput, bool) {
	if span == nil {
		return gcpTraceSpanInput{}, false
	}
	startTime := span.GetStartTime()
	endTime := span.GetEndTime()

	displayName := ""
	if span.GetDisplayName() != nil {
		displayName = strings.TrimSpace(span.GetDisplayName().GetValue())
	}
	sameProcess := false
	var sameProcessPtr *bool
	if span.GetSameProcessAsParentSpan() != nil {
		sameProcess = span.GetSameProcessAsParentSpan().GetValue()
		sameProcessPtr = &sameProcess
	}

	return gcpTraceSpanInput{
		Name:        strings.TrimSpace(span.GetName()),
		SpanID:      strings.TrimSpace(span.GetSpanId()),
		ParentSpan:  strings.TrimSpace(span.GetParentSpanId()),
		DisplayName: displayName,
		StartTime:   startTime.AsTime(),
		EndTime:     endTime.AsTime(),
		HasStart:    startTime != nil,
		HasEnd:      endTime != nil,
		SpanKind:    span.GetSpanKind().String(),
		SameProcess: sameProcessPtr,
	}, true
}

func gcpTraceSpanProtoFixture(span gcpTraceNormalizedSpan) *tracepb.Span {
	out := &tracepb.Span{
		Name:        span.Name,
		SpanId:      span.SpanID,
		DisplayName: &tracepb.TruncatableString{Value: span.DisplayName},
		StartTime:   timestamppb.New(span.StartTime),
		EndTime:     timestamppb.New(span.EndTime),
		SpanKind:    gcpTraceSpanKindToProto(span.SpanKind),
		Attributes: &tracepb.Span_Attributes{
			AttributeMap: map[string]*tracepb.AttributeValue{
				"stackyard.example": {
					Value: &tracepb.AttributeValue_StringValue{
						StringValue: &tracepb.TruncatableString{Value: "trace"},
					},
				},
			},
			DroppedAttributesCount: 0,
		},
		SameProcessAsParentSpan: wrapperspb.Bool(span.SameProcess),
	}
	if span.ParentSpan != "" {
		out.ParentSpanId = span.ParentSpan
	}
	return out
}

func gcpTraceSpanKindToProto(kind string) tracepb.Span_SpanKind {
	switch gcpTraceNormalizeSpanKind(kind) {
	case "SPAN_KIND_INTERNAL":
		return tracepb.Span_INTERNAL
	case "SPAN_KIND_CLIENT":
		return tracepb.Span_CLIENT
	case "SPAN_KIND_SERVER":
		return tracepb.Span_SERVER
	case "SPAN_KIND_PRODUCER":
		return tracepb.Span_PRODUCER
	case "SPAN_KIND_CONSUMER":
		return tracepb.Span_CONSUMER
	default:
		return tracepb.Span_SPAN_KIND_UNSPECIFIED
	}
}

func gcpTraceGRPCValidationError(code, message string) ([]byte, string, string, bool) {
	reason := gcpTraceValidationReason(message)
	switch code {
	case "NotFound":
		return grpcNotFound(reason)
	case "FailedPrecondition":
		return grpcFailedPrecondition(reason)
	default:
		return grpcInvalidArgument(reason)
	}
}

func gcpTraceValidationReason(message string) string {
	switch {
	case strings.Contains(message, "name is required"):
		return "name-required"
	case strings.Contains(message, "name project must match"):
		return "name-project-mismatch"
	case strings.Contains(message, "name trace must match"):
		return "name-trace-mismatch"
	case strings.Contains(message, "name span must match"):
		return "name-span-mismatch"
	case strings.Contains(message, "project is invalid"):
		return "project-invalid"
	case strings.Contains(message, "trace id is invalid"):
		return "trace-id-invalid"
	case strings.Contains(message, "span id is invalid"):
		return "span-id-invalid"
	case strings.Contains(message, "spanId is required"):
		return "span_id-required"
	case strings.Contains(message, "spanId is invalid"):
		return "span_id-invalid"
	case strings.Contains(message, "spanId must match"):
		return "span_id-mismatch"
	case strings.Contains(message, "parentSpanId is invalid"):
		return "parent_span_id-invalid"
	case strings.Contains(message, "displayName.value is required"):
		return "display_name-required"
	case strings.Contains(message, "displayName.value must be"):
		return "display_name-too-long"
	case strings.Contains(message, "startTime is required"):
		return "start_time-required"
	case strings.Contains(message, "endTime is required"):
		return "end_time-required"
	case strings.Contains(message, "endTime must be >= startTime"):
		return "time-range-invalid"
	case strings.Contains(message, "duration exceeds staged limit"):
		return "span-duration-exceeds-staged-limit"
	case strings.Contains(message, "spanKind is invalid"):
		return "span_kind-invalid"
	default:
		return "invalid-argument"
	}
}
