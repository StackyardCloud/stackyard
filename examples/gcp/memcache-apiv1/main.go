package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	memcache "cloud.google.com/go/memcache/apiv1"
	"cloud.google.com/go/memcache/apiv1/memcachepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *memcache.CloudMemcacheClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	instanceID := getenv("STACKYARD_GCP_MEMCACHE_INSTANCE_ID", "cache-a")
	operationID := getenv("STACKYARD_GCP_MEMCACHE_OPERATION_ID", "op-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	parent := projectName + "/locations/" + locationID
	instanceName := parent + "/instances/" + instanceID
	operationName := parent + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Cloud Memorystore (Memcache) apiv1 client using %s\n", apiEndpoint)

	client, err := memcache.NewCloudMemcacheRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloud memcache client: %v", err)
	}
	defer closeClient("cloud memcache", client.Close)

	calls := []callSpec{
		{
			name: "ListInstances",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				it := c.ListInstances(ctx, &memcachepb.ListInstancesRequest{
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
			name: "GetInstance",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				_, err := c.GetInstance(ctx, &memcachepb.GetInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "CreateInstance",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				_, err := c.CreateInstance(ctx, &memcachepb.CreateInstanceRequest{
					Parent:     parent,
					InstanceId: instanceID,
					Instance: &memcachepb.Instance{
						Name:      instanceName,
						NodeCount: 1,
						NodeConfig: &memcachepb.Instance_NodeConfig{
							CpuCount:     1,
							MemorySizeMb: 1024,
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateInstance",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				_, err := c.UpdateInstance(ctx, &memcachepb.UpdateInstanceRequest{
					Instance: &memcachepb.Instance{
						Name:        instanceName,
						DisplayName: "stackyard cache",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "UpdateParameters",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				_, err := c.UpdateParameters(ctx, &memcachepb.UpdateParametersRequest{
					Name:       instanceName,
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"params.max_item_size"}},
					Parameters: &memcachepb.MemcacheParameters{
						Params: map[string]string{"max_item_size": "2m"},
					},
				})
				return err
			},
		},
		{
			name: "ApplyParameters",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				_, err := c.ApplyParameters(ctx, &memcachepb.ApplyParametersRequest{
					Name:     instanceName,
					ApplyAll: true,
				})
				return err
			},
		},
		{
			name: "RescheduleMaintenance",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				_, err := c.RescheduleMaintenance(ctx, &memcachepb.RescheduleMaintenanceRequest{
					Instance:       instanceName,
					RescheduleType: memcachepb.RescheduleMaintenanceRequest_IMMEDIATE,
				})
				return err
			},
		},
		{
			name: "DeleteInstance",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				_, err := c.DeleteInstance(ctx, &memcachepb.DeleteInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
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
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
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
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *memcache.CloudMemcacheClient) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
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

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
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
