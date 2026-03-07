package server

import (
	"net/http"
	"strings"
	"testing"
	"time"

	tracepb "cloud.google.com/go/trace/apiv2/tracepb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestGCPStage4GRPCParity_Trace(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	traceID := "0123456789abcdef0123456789abcdef"
	spanID := "1111111111111111"
	spanName := "projects/stackyard/traces/" + traceID + "/spans/" + spanID

	restCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/"+spanName, []byte(`{
		"spanId":"`+spanID+`",
		"displayName":{"value":"stackyard.trace.parity"},
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-01T00:00:01Z"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace",
	})
	if restCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest trace create span, got %d body=%s", restCreateResp.StatusCode, string(providerContractBody(t, restCreateResp)))
	}
	restCreateBody := providerContractJSONMap(t, restCreateResp)
	restName, _ := restCreateBody["name"].(string)
	if strings.TrimSpace(restName) == "" {
		t.Fatalf("expected rest span name in create response")
	}

	var grpcCreateResp tracepb.Span
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpTraceCreateSpanMethod, &tracepb.Span{
		Name:        spanName,
		SpanId:      spanID,
		DisplayName: &tracepb.TruncatableString{Value: "stackyard.trace.parity"},
		StartTime:   timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
		EndTime:     timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 1, 0, time.UTC)),
		SpanKind:    tracepb.Span_SERVER,
		SameProcessAsParentSpan: &wrapperspb.BoolValue{
			Value: true,
		},
	}, &grpcCreateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for create span, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcCreateResp.GetName() != restName {
		t.Fatalf("expected grpc create span name %q to match rest %q", grpcCreateResp.GetName(), restName)
	}

	restBatchResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/traces:batchWrite", []byte(`{
		"spans":[{
			"name":"`+spanName+`",
			"spanId":"`+spanID+`",
			"displayName":{"value":"stackyard.trace.batch"},
			"startTime":"2026-01-01T00:00:00Z",
			"endTime":"2026-01-01T00:00:01Z"
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace",
	})
	if restBatchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest trace batchWrite, got %d body=%s", restBatchResp.StatusCode, string(providerContractBody(t, restBatchResp)))
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTraceBatchWriteSpansMethod, &tracepb.BatchWriteSpansRequest{
		Name: "projects/stackyard",
		Spans: []*tracepb.Span{
			{
				Name:        spanName,
				SpanId:      spanID,
				DisplayName: &tracepb.TruncatableString{Value: "stackyard.trace.batch"},
				StartTime:   timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
				EndTime:     timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 1, 0, time.UTC)),
				SpanKind:    tracepb.Span_SERVER,
			},
		},
	}, nil)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for batchWrite, got %q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTraceCreateSpanMethod, &tracepb.Span{
		Name:      spanName,
		SpanId:    spanID,
		StartTime: timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
		EndTime:   timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 1, 0, time.UTC)),
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "display_name-required") {
		t.Fatalf("expected grpc invalid argument for missing displayName, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTraceCreateSpanMethod, &tracepb.Span{
		Name:        spanName,
		SpanId:      spanID,
		DisplayName: &tracepb.TruncatableString{Value: "stackyard.trace.duration"},
		StartTime:   timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
		EndTime:     timestamppb.New(time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)),
	}, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "span-duration-exceeds-staged-limit") {
		t.Fatalf("expected grpc failed precondition for long span duration, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTraceCreateSpanMethod, &tracepb.Span{
		Name:        "projects/missing-project/traces/0123456789abcdef0123456789abcdef/spans/1111111111111111",
		SpanId:      "1111111111111111",
		DisplayName: &tracepb.TruncatableString{Value: "stackyard.trace.missing"},
		StartTime:   timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
		EndTime:     timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 1, 0, time.UTC)),
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "project-not-found") {
		t.Fatalf("expected grpc not found for missing project, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
