package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	apihub "cloud.google.com/go/apihub/apiv1"
	"cloud.google.com/go/apihub/apiv1/apihubpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *apihub.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	location := getenv("STACKYARD_GCP_LOCATION", "projects/stackyard/locations/us-central1")
	apiID := getenv("STACKYARD_GCP_API_ID", "team-api")
	versionID := getenv("STACKYARD_GCP_VERSION_ID", "v1")
	definitionID := getenv("STACKYARD_GCP_DEFINITION_ID", "openapi")
	operationID := getenv("STACKYARD_GCP_OPERATION_ID", "get-orders")

	apiName := location + "/apis/" + apiID
	versionName := apiName + "/versions/" + versionID
	definitionName := versionName + "/definitions/" + definitionID
	operationName := versionName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP API Hub apiv1 client using %s\n", apiEndpoint)

	client, err := apihub.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create apihub client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListAttributes",
			call: func(ctx context.Context, c *apihub.Client) error {
				it := c.ListAttributes(ctx, &apihubpb.ListAttributesRequest{
					Parent:   location,
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
			name: "ListDeployments",
			call: func(ctx context.Context, c *apihub.Client) error {
				it := c.ListDeployments(ctx, &apihubpb.ListDeploymentsRequest{
					Parent:   location,
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
			name: "ListExternalApis",
			call: func(ctx context.Context, c *apihub.Client) error {
				it := c.ListExternalApis(ctx, &apihubpb.ListExternalApisRequest{
					Parent:   location,
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
			name: "ListVersions",
			call: func(ctx context.Context, c *apihub.Client) error {
				it := c.ListVersions(ctx, &apihubpb.ListVersionsRequest{
					Parent:   apiName,
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
			name: "GetDefinition",
			call: func(ctx context.Context, c *apihub.Client) error {
				_, err := c.GetDefinition(ctx, &apihubpb.GetDefinitionRequest{Name: definitionName})
				return err
			},
		},
		{
			name: "GetApiOperation",
			call: func(ctx context.Context, c *apihub.Client) error {
				_, err := c.GetApiOperation(ctx, &apihubpb.GetApiOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "SearchResources",
			call: func(ctx context.Context, c *apihub.Client) error {
				it := c.SearchResources(ctx, &apihubpb.SearchResourcesRequest{
					Location: location,
					Query:    "orders",
					PageSize: 1,
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
		fmt.Fprintf(os.Stderr, "warning: close apihub client: %v\n", err)
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
