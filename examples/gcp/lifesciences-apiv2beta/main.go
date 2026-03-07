package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	lifesciences "cloud.google.com/go/lifesciences/apiv2beta"
	"cloud.google.com/go/lifesciences/apiv2beta/lifesciencespb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *lifesciences.WorkflowsServiceV2BetaClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	operationID := getenv("STACKYARD_GCP_LIFESCIENCES_OPERATION_ID", "op-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Life Sciences apiv2beta client using %s\n", apiEndpoint)

	client, err := lifesciences.NewWorkflowsServiceV2BetaRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create lifesciences client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "RunPipeline",
			call: func(ctx context.Context, c *lifesciences.WorkflowsServiceV2BetaClient) error {
				_, err := c.RunPipeline(ctx, &lifesciencespb.RunPipelineRequest{
					Parent: projectName + "/locations/" + locationID,
					Pipeline: &lifesciencespb.Pipeline{
						Actions: []*lifesciencespb.Action{
							{ImageUri: "alpine:latest"},
						},
					},
					Labels: map[string]string{"stackyard": "true"},
				})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *lifesciences.WorkflowsServiceV2BetaClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *lifesciences.WorkflowsServiceV2BetaClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *lifesciences.WorkflowsServiceV2BetaClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *lifesciences.WorkflowsServiceV2BetaClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *lifesciences.WorkflowsServiceV2BetaClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
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
		fmt.Fprintf(os.Stderr, "warning: close lifesciences client: %v\n", err)
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
