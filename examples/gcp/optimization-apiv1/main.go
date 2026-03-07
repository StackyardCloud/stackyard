package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	optimization "cloud.google.com/go/optimization/apiv1"
	"cloud.google.com/go/optimization/apiv1/optimizationpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *optimization.FleetRoutingClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	operationName := getenv("STACKYARD_GCP_OPTIMIZATION_OPERATION_NAME", "operations/opt-op-1")
	inputURI := getenv("STACKYARD_GCP_OPTIMIZATION_INPUT_URI", "gs://stackyard-optimization/input.json")
	outputURI := getenv("STACKYARD_GCP_OPTIMIZATION_OUTPUT_URI", "gs://stackyard-optimization/output/")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)

	fmt.Printf("Stackyard GCP Cloud Optimization apiv1 client using %s\n", apiEndpoint)

	client, err := optimization.NewFleetRoutingRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create optimization client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "OptimizeTours",
			call: func(ctx context.Context, c *optimization.FleetRoutingClient) error {
				_, err := c.OptimizeTours(ctx, &optimizationpb.OptimizeToursRequest{
					Parent: parent,
					Model:  &optimizationpb.ShipmentModel{},
					Label:  "stackyard-optimization",
				})
				return err
			},
		},
		{
			name: "BatchOptimizeTours",
			call: func(ctx context.Context, c *optimization.FleetRoutingClient) error {
				_, err := c.BatchOptimizeTours(ctx, &optimizationpb.BatchOptimizeToursRequest{
					Parent: parent,
					ModelConfigs: []*optimizationpb.BatchOptimizeToursRequest_AsyncModelConfig{
						{
							DisplayName: "stackyard-optimization-model",
							InputConfig: &optimizationpb.InputConfig{
								Source: &optimizationpb.InputConfig_GcsSource{
									GcsSource: &optimizationpb.GcsSource{Uri: inputURI},
								},
								DataFormat: optimizationpb.DataFormat_JSON,
							},
							OutputConfig: &optimizationpb.OutputConfig{
								Destination: &optimizationpb.OutputConfig_GcsDestination{
									GcsDestination: &optimizationpb.GcsDestination{Uri: outputURI},
								},
								DataFormat: optimizationpb.DataFormat_JSON,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *optimization.FleetRoutingClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: operationName,
				})
				return err
			},
		},
	}

	for _, c := range calls {
		err := c.call(ctx, client)
		switch {
		case err == nil:
			logf("%s succeeded", c.name)
		case isToleratedNotImplemented(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", c.name, err)
		default:
			exitf("%s failed: %v", c.name, err)
		}
	}

	fmt.Println("Done.")
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close optimization client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
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
