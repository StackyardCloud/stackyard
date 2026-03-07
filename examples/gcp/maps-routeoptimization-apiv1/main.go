package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	routeoptimization "cloud.google.com/go/maps/routeoptimization/apiv1"
	"cloud.google.com/go/maps/routeoptimization/apiv1/routeoptimizationpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *routeoptimization.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	operationName := getenv("STACKYARD_GCP_ROUTEOPT_OPERATION_NAME", "operations/routeopt-op-1")
	inputURI := getenv("STACKYARD_GCP_ROUTEOPT_INPUT_URI", "gs://stackyard-routeopt/input.json")
	outputURI := getenv("STACKYARD_GCP_ROUTEOPT_OUTPUT_URI", "gs://stackyard-routeopt/output/")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)

	fmt.Printf("Stackyard GCP Maps Route Optimization apiv1 client using %s\n", apiEndpoint)

	client, err := routeoptimization.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create routeoptimization client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "OptimizeTours",
			call: func(ctx context.Context, c *routeoptimization.Client) error {
				_, err := c.OptimizeTours(ctx, &routeoptimizationpb.OptimizeToursRequest{
					Parent: parent,
					Model:  &routeoptimizationpb.ShipmentModel{},
					Label:  "stackyard-routeoptimization",
				})
				return err
			},
		},
		{
			name: "BatchOptimizeTours",
			call: func(ctx context.Context, c *routeoptimization.Client) error {
				_, err := c.BatchOptimizeTours(ctx, &routeoptimizationpb.BatchOptimizeToursRequest{
					Parent: parent,
					ModelConfigs: []*routeoptimizationpb.BatchOptimizeToursRequest_AsyncModelConfig{
						{
							DisplayName: "stackyard-routeopt-model",
							InputConfig: &routeoptimizationpb.InputConfig{
								Source: &routeoptimizationpb.InputConfig_GcsSource{
									GcsSource: &routeoptimizationpb.GcsSource{Uri: inputURI},
								},
								DataFormat: routeoptimizationpb.DataFormat_JSON,
							},
							OutputConfig: &routeoptimizationpb.OutputConfig{
								Destination: &routeoptimizationpb.OutputConfig_GcsDestination{
									GcsDestination: &routeoptimizationpb.GcsDestination{Uri: outputURI},
								},
								DataFormat: routeoptimizationpb.DataFormat_JSON,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *routeoptimization.Client) error {
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
			logf("%s returned NotImplemented (expected in staged emulation)", c.name)
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

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close routeoptimization client: %v\n", err)
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
