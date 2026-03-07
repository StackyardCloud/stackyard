package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	parallelstore "cloud.google.com/go/parallelstore/apiv1"
	"cloud.google.com/go/parallelstore/apiv1/parallelstorepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	longrunningpb "google.golang.org/genproto/googleapis/longrunning"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *parallelstore.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1-c")
	instanceID := getenv("STACKYARD_GCP_PARALLELSTORE_INSTANCE_ID", "instance-a")
	network := getenv("STACKYARD_GCP_PARALLELSTORE_NETWORK", fmt.Sprintf("projects/%s/global/networks/default", projectID))
	gcsURI := getenv("STACKYARD_GCP_PARALLELSTORE_GCS_URI", "gs://stackyard-parallelstore-transfer/")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	instanceName := fmt.Sprintf("%s/instances/%s", parent, instanceID)
	operationName := fmt.Sprintf("%s/operations/op-1", parent)

	fmt.Printf("Stackyard GCP Parallelstore apiv1 client using %s\n", apiEndpoint)

	client, err := parallelstore.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create parallelstore client: %v", err)
	}
	defer closeClient("parallelstore", client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListInstances",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				it := c.ListInstances(ctx, &parallelstorepb.ListInstancesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetInstance",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				_, err := c.GetInstance(ctx, &parallelstorepb.GetInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "CreateInstance",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				_, err := c.CreateInstance(ctx, &parallelstorepb.CreateInstanceRequest{
					Parent:     parent,
					InstanceId: instanceID,
					Instance: &parallelstorepb.Instance{
						Name:        instanceName,
						Description: "stackyard parallelstore instance",
						CapacityGib: 12000,
						Network:     network,
						Labels:      map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "UpdateInstance",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				_, err := c.UpdateInstance(ctx, &parallelstorepb.UpdateInstanceRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
					Instance: &parallelstorepb.Instance{
						Name:        instanceName,
						Description: "stackyard parallelstore instance updated",
					},
				})
				return err
			},
		},
		{
			name: "ExportData",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				_, err := c.ExportData(ctx, &parallelstorepb.ExportDataRequest{
					Name: instanceName,
					Source: &parallelstorepb.ExportDataRequest_SourceParallelstore{
						SourceParallelstore: &parallelstorepb.SourceParallelstore{Path: "/"},
					},
					Destination: &parallelstorepb.ExportDataRequest_DestinationGcsBucket{
						DestinationGcsBucket: &parallelstorepb.DestinationGcsBucket{Uri: gcsURI},
					},
				})
				return err
			},
		},
		{
			name: "ImportData",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				_, err := c.ImportData(ctx, &parallelstorepb.ImportDataRequest{
					Name: instanceName,
					Source: &parallelstorepb.ImportDataRequest_SourceGcsBucket{
						SourceGcsBucket: &parallelstorepb.SourceGcsBucket{Uri: gcsURI},
					},
					Destination: &parallelstorepb.ImportDataRequest_DestinationParallelstore{
						DestinationParallelstore: &parallelstorepb.DestinationParallelstore{Path: "/"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteInstance",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				_, err := c.DeleteInstance(ctx, &parallelstorepb.DeleteInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *parallelstore.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
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

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
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

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "notimplemented") || strings.Contains(lower, "not implemented")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
