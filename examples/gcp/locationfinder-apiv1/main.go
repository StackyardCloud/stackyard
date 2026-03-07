package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	locationfinder "cloud.google.com/go/locationfinder/apiv1"
	"cloud.google.com/go/locationfinder/apiv1/locationfinderpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *locationfinder.CloudLocationFinderClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	cloudLocationID := getenv("STACKYARD_GCP_LOCATIONFINDER_CLOUD_LOCATION_ID", "us-east1")
	sourceCloudLocationID := getenv("STACKYARD_GCP_LOCATIONFINDER_SOURCE_CLOUD_LOCATION_ID", "us-central1")
	searchQuery := getenv("STACKYARD_GCP_LOCATIONFINDER_QUERY", "latency")

	projectName := fmt.Sprintf("projects/%s", projectID)
	parent := projectName + "/locations/" + locationID
	cloudLocationName := parent + "/cloudLocations/" + cloudLocationID
	sourceCloudLocation := parent + "/cloudLocations/" + sourceCloudLocationID

	fmt.Printf("Stackyard GCP Cloud Location Finder apiv1 client using %s\n", apiEndpoint)

	client, err := locationfinder.NewCloudLocationFinderRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create locationfinder client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListCloudLocations",
			call: func(ctx context.Context, c *locationfinder.CloudLocationFinderClient) error {
				it := c.ListCloudLocations(ctx, &locationfinderpb.ListCloudLocationsRequest{
					Parent:   parent,
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
			name: "GetCloudLocation",
			call: func(ctx context.Context, c *locationfinder.CloudLocationFinderClient) error {
				_, err := c.GetCloudLocation(ctx, &locationfinderpb.GetCloudLocationRequest{
					Name: cloudLocationName,
				})
				return err
			},
		},
		{
			name: "SearchCloudLocations",
			call: func(ctx context.Context, c *locationfinder.CloudLocationFinderClient) error {
				it := c.SearchCloudLocations(ctx, &locationfinderpb.SearchCloudLocationsRequest{
					Parent:              parent,
					SourceCloudLocation: sourceCloudLocation,
					Query:               searchQuery,
					PageSize:            1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *locationfinder.CloudLocationFinderClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *locationfinder.CloudLocationFinderClient) error {
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
		fmt.Fprintf(os.Stderr, "warning: close locationfinder client: %v\n", err)
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
