package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	trace "cloud.google.com/go/trace/apiv2"
	tracepb "cloud.google.com/go/trace/apiv2/tracepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
	spanID := getenv("STACKYARD_GCP_SPAN_ID", "1111111111111111")
	batchSpanID := getenv("STACKYARD_GCP_BATCH_SPAN_ID", "2222222222222222")

	spanName := fmt.Sprintf("projects/%s/traces/%s/spans/%s", projectID, traceID, spanID)
	batchSpanName := fmt.Sprintf("projects/%s/traces/%s/spans/%s", projectID, traceID, batchSpanID)

	fmt.Printf("Stackyard GCP Stackdriver Trace V2 trace/apiv2 client using %s (grpc=%s)\n", apiEndpoint, grpcEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	client, err := trace.NewClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create trace client: %v", err)
	}
	defer closeClient(client.Close)

	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(1 * time.Second)

	calls := []callSpec{
		{
			name: "CreateSpan",
			call: func(ctx context.Context, c *trace.Client) error {
				span, err := c.CreateSpan(ctx, &tracepb.Span{
					Name:        spanName,
					SpanId:      spanID,
					DisplayName: &tracepb.TruncatableString{Value: "stackyard.trace.create"},
					StartTime:   timestamppb.New(start),
					EndTime:     timestamppb.New(end),
					SpanKind:    tracepb.Span_SERVER,
					SameProcessAsParentSpan: &wrapperspb.BoolValue{
						Value: true,
					},
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(span.GetName()) == "" {
					return errors.New("CreateSpan returned empty span name")
				}
				return nil
			},
		},
		{
			name: "BatchWriteSpans",
			call: func(ctx context.Context, c *trace.Client) error {
				return c.BatchWriteSpans(ctx, &tracepb.BatchWriteSpansRequest{
					Name: fmt.Sprintf("projects/%s", projectID),
					Spans: []*tracepb.Span{
						{
							Name:        batchSpanName,
							SpanId:      batchSpanID,
							DisplayName: &tracepb.TruncatableString{Value: "stackyard.trace.batch"},
							StartTime:   timestamppb.New(start),
							EndTime:     timestamppb.New(end),
							SpanKind:    tracepb.Span_SERVER,
						},
					},
				})
			},
		},
	}

	for _, call := range calls {
		if err := call.call(ctx, client); err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		logf("%s succeeded", call.name)
	}

	_, err = client.CreateSpan(ctx, &tracepb.Span{
		Name:      spanName,
		SpanId:    spanID,
		StartTime: timestamppb.New(start),
		EndTime:   timestamppb.New(end),
	})
	if err == nil {
		exitf("CreateSpan validation call unexpectedly succeeded")
	}
	if !isExpectedInvalidArgument(err) {
		exitf("CreateSpan validation call returned unexpected error: %v", err)
	}
	logf("CreateSpan validation call returned InvalidArgument (expected)")

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, apiEndpoint string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	probeURL := fmt.Sprintf("%s/v1/projects/stackyard/locations/us-central1/trace?stackyard_contract_probe=1&typedSuccess=1", apiEndpoint)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "trace")

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("trace contract probe did not become ready: %s", probeURL)
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
		fmt.Fprintf(os.Stderr, "warning: close trace client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
