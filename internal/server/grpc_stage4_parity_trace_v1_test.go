package server

import (
	"net/http"
	"strings"
	"testing"
	"time"

	tracev1pb "cloud.google.com/go/trace/apiv1/tracepb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGCPStage4GRPCParity_TraceV1(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	traceID := "0123456789abcdef0123456789abcdef"

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/traces?pageSize=1&view=COMPLETE", nil, map[string]string{
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest trace_v1 list traces, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restTraces, ok := restListBody["traces"].([]any)
	if !ok || len(restTraces) == 0 {
		t.Fatalf("expected traces list in rest payload, got %#v", restListBody["traces"])
	}
	restTrace, _ := restTraces[0].(map[string]any)
	restTraceID, _ := restTrace["traceId"].(string)

	var grpcListResp tracev1pb.ListTracesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpTraceV1ListTracesMethod, &tracev1pb.ListTracesRequest{
		ProjectId: "stackyard",
		PageSize:  1,
		View:      tracev1pb.ListTracesRequest_COMPLETE,
	}, &grpcListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for trace_v1 list traces, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcListResp.GetTraces()) != 1 {
		t.Fatalf("expected one grpc trace in list, got %d", len(grpcListResp.GetTraces()))
	}
	if grpcListResp.GetTraces()[0].GetTraceId() != restTraceID {
		t.Fatalf("expected grpc trace id %q to match rest %q", grpcListResp.GetTraces()[0].GetTraceId(), restTraceID)
	}

	restGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/traces/"+traceID, nil, map[string]string{
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if restGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest trace_v1 get trace, got %d body=%s", restGetResp.StatusCode, string(providerContractBody(t, restGetResp)))
	}
	restGetBody := providerContractJSONMap(t, restGetResp)
	restGetTraceID, _ := restGetBody["traceId"].(string)

	var grpcGetResp tracev1pb.Trace
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTraceV1GetTraceMethod, &tracev1pb.GetTraceRequest{
		ProjectId: "stackyard",
		TraceId:   traceID,
	}, &grpcGetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for trace_v1 get trace, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcGetResp.GetTraceId() != restGetTraceID {
		t.Fatalf("expected grpc trace id %q to match rest %q", grpcGetResp.GetTraceId(), restGetTraceID)
	}

	restPatchResp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/traces", []byte(`{
		"traces":[{
			"traceId":"0123456789abcdef0123456789abcdef",
			"spans":[{
				"spanId":"9001",
				"kind":1,
				"name":"/stackyard/tracev1/parity",
				"startTime":"2026-01-01T00:00:00Z",
				"endTime":"2026-01-01T00:00:01Z"
			}]
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if restPatchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest trace_v1 patch traces, got %d body=%s", restPatchResp.StatusCode, string(providerContractBody(t, restPatchResp)))
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTraceV1PatchTracesMethod, &tracev1pb.PatchTracesRequest{
		ProjectId: "stackyard",
		Traces: &tracev1pb.Traces{
			Traces: []*tracev1pb.Trace{
				{
					ProjectId: "stackyard",
					TraceId:   "0123456789abcdef0123456789abcdef",
					Spans: []*tracev1pb.TraceSpan{
						{
							SpanId:    9001,
							Kind:      tracev1pb.TraceSpan_RPC_SERVER,
							Name:      "/stackyard/tracev1/parity",
							StartTime: timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
							EndTime:   timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 1, 0, time.UTC)),
						},
					},
				},
			},
		},
	}, nil)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for trace_v1 patch traces, got %q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTraceV1GetTraceMethod, &tracev1pb.GetTraceRequest{
		ProjectId: "stackyard",
		TraceId:   "not-a-trace-id",
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "trace_id-invalid") {
		t.Fatalf("expected grpc invalid argument for trace_v1 get invalid trace_id, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTraceV1GetTraceMethod, &tracev1pb.GetTraceRequest{
		ProjectId: "stackyard",
		TraceId:   "ffffffffffffffffffffffffffffffff",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "trace-not-found") {
		t.Fatalf("expected grpc not found for trace_v1 get missing trace, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTraceV1PatchTracesMethod, &tracev1pb.PatchTracesRequest{
		ProjectId: "stackyard",
		Traces: &tracev1pb.Traces{
			Traces: []*tracev1pb.Trace{
				{
					TraceId: "0123456789abcdef0123456789abcdef",
					Spans: []*tracev1pb.TraceSpan{
						{
							SpanId:    9002,
							Kind:      tracev1pb.TraceSpan_RPC_SERVER,
							Name:      "/stackyard/tracev1/long-duration",
							StartTime: timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
							EndTime:   timestamppb.New(time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)),
						},
					},
				},
			},
		},
	}, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "span-duration-exceeds-staged-limit") {
		t.Fatalf("expected grpc failed precondition for trace_v1 patch long span, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
