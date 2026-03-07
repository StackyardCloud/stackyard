package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	ids "cloud.google.com/go/ids/apiv1"
	"cloud.google.com/go/ids/apiv1/idspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *ids.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	endpointID := getenv("STACKYARD_GCP_IDS_ENDPOINT_ID", "ids-endpoint-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	endpointName := locationName + "/endpoints/" + endpointID
	networkName := fmt.Sprintf("projects/%s/global/networks/default", projectID)

	fmt.Printf("Stackyard GCP Cloud IDS apiv1 client using %s\n", apiEndpoint)

	client, err := ids.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create ids client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListEndpoints",
			call: func(ctx context.Context, c *ids.Client) error {
				it := c.ListEndpoints(ctx, &idspb.ListEndpointsRequest{
					Parent:   locationName,
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
			name: "GetEndpoint",
			call: func(ctx context.Context, c *ids.Client) error {
				_, err := c.GetEndpoint(ctx, &idspb.GetEndpointRequest{Name: endpointName})
				return err
			},
		},
		{
			name: "CreateEndpoint",
			call: func(ctx context.Context, c *ids.Client) error {
				_, err := c.CreateEndpoint(ctx, &idspb.CreateEndpointRequest{
					Parent:     locationName,
					EndpointId: endpointID,
					Endpoint: &idspb.Endpoint{
						Network:  networkName,
						Severity: idspb.Endpoint_HIGH,
					},
				})
				return err
			},
		},
		{
			name: "DeleteEndpoint",
			call: func(ctx context.Context, c *ids.Client) error {
				_, err := c.DeleteEndpoint(ctx, &idspb.DeleteEndpointRequest{Name: endpointName})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
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
		fmt.Fprintf(os.Stderr, "warning: close ids client: %v\n", err)
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
