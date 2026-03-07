package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	trace "cloud.google.com/go/trace/apiv1"
	tracepb "cloud.google.com/go/trace/apiv1/tracepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type callSpec struct {
	name string
	call func(context.Context, *trace.Client) error
}

func main() {
	ctx := context.Background()

	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	apiEndpoint := endpoint + "/gcp"
	grpcEndpoint := strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	traceID := getenv("STACKYARD_GCP_TRACE_ID", "0123456789abcdef0123456789abcdef")
	rootSpanID := getenvUint64("STACKYARD_GCP_TRACE_ROOT_SPAN_ID", 9001)
	childSpanID := getenvUint64("STACKYARD_GCP_TRACE_CHILD_SPAN_ID", 9002)

	fmt.Printf("Stackyard GCP Stackdriver Trace V1 trace/apiv1 client using %s (grpc=%s)\n", apiEndpoint, grpcEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	client, err := trace.NewClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create trace v1 client: %v", err)
	}
	defer closeClient(client.Close)

	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(900 * time.Millisecond)

	calls := []callSpec{
		{
			name: "PatchTraces",
			call: func(ctx context.Context, c *trace.Client) error {
				return c.PatchTraces(ctx, &tracepb.PatchTracesRequest{
					ProjectId: projectID,
					Traces: &tracepb.Traces{
						Traces: []*tracepb.Trace{
							{
								ProjectId: projectID,
								TraceId:   traceID,
								Spans: []*tracepb.TraceSpan{
									{
										SpanId:    rootSpanID,
										Kind:      tracepb.TraceSpan_RPC_SERVER,
										Name:      "/stackyard/trace/apiv1/root",
										StartTime: timestamppb.New(start),
										EndTime:   timestamppb.New(end),
										Labels: map[string]string{
											"/component": "trace_v1",
											"/http/path": "/stackyard/trace/apiv1/root",
										},
									},
									{
										SpanId:       childSpanID,
										ParentSpanId: rootSpanID,
										Kind:         tracepb.TraceSpan_RPC_CLIENT,
										Name:         "/stackyard/trace/apiv1/downstream",
										StartTime:    timestamppb.New(start.Add(100 * time.Millisecond)),
										EndTime:      timestamppb.New(start.Add(400 * time.Millisecond)),
										Labels: map[string]string{
											"/component": "trace_v1_child",
										},
									},
								},
							},
						},
					},
				})
			},
		},
		{
			name: "GetTrace",
			call: func(ctx context.Context, c *trace.Client) error {
				traceResp, err := c.GetTrace(ctx, &tracepb.GetTraceRequest{
					ProjectId: projectID,
					TraceId:   traceID,
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(traceResp.GetTraceId()) == "" {
					return errors.New("GetTrace returned empty trace_id")
				}
				return nil
			},
		},
		{
			name: "ListTraces",
			call: func(ctx context.Context, c *trace.Client) error {
				it := c.ListTraces(ctx, &tracepb.ListTracesRequest{
					ProjectId: projectID,
					PageSize:  1,
					View:      tracepb.ListTracesRequest_COMPLETE,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
	}

	for _, call := range calls {
		if err := call.call(ctx, client); err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		logf("%s succeeded", call.name)
	}

	_, err = client.GetTrace(ctx, &tracepb.GetTraceRequest{
		ProjectId: projectID,
		TraceId:   "not-a-trace-id",
	})
	if err == nil {
		exitf("GetTrace validation call unexpectedly succeeded")
	}
	if !isExpectedInvalidArgument(err) {
		exitf("GetTrace validation call returned unexpected error: %v", err)
	}
	logf("GetTrace validation call returned InvalidArgument (expected)")

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, apiEndpoint string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	probeURL := fmt.Sprintf("%s/v1/projects/stackyard/locations/us-central1/trace_v1?stackyard_contract_probe=1&typedSuccess=1", apiEndpoint)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "trace_v1")

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("trace_v1 contract probe did not become ready: %s", probeURL)
}

func isExpectedInvalidArgument(err error) bool {
	if err == nil {
		return false
	}
	if grpcStatus, ok := status.FromError(err); ok {
		return grpcStatus.Code() == codes.InvalidArgument
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusBadRequest && strings.Contains(strings.ToLower(apiErr.Message), "invalidargument")
	}
	return false
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close trace v1 client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getenvUint64(key string, fallback uint64) uint64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
