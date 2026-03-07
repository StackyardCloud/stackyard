package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	datafusion "cloud.google.com/go/datafusion/apiv1"
	"cloud.google.com/go/datafusion/apiv1/datafusionpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *datafusion.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	instanceID := getenv("STACKYARD_GCP_DATAFUSION_INSTANCE_ID", "team-instance")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	instanceName := locationName + "/instances/" + instanceID

	fmt.Printf("Stackyard GCP Data Fusion apiv1 client using %s\n", apiEndpoint)

	client, err := datafusion.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create datafusion client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListAvailableVersions",
			call: func(ctx context.Context, c *datafusion.Client) error {
				it := c.ListAvailableVersions(ctx, &datafusionpb.ListAvailableVersionsRequest{
					Parent:          locationName,
					PageSize:        1,
					LatestPatchOnly: true,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "ListInstances",
			call: func(ctx context.Context, c *datafusion.Client) error {
				it := c.ListInstances(ctx, &datafusionpb.ListInstancesRequest{
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
			name: "GetInstance",
			call: func(ctx context.Context, c *datafusion.Client) error {
				_, err := c.GetInstance(ctx, &datafusionpb.GetInstanceRequest{
					Name: instanceName,
				})
				return err
			},
		},
		{
			name: "CreateInstance",
			call: func(ctx context.Context, c *datafusion.Client) error {
				_, err := c.CreateInstance(ctx, &datafusionpb.CreateInstanceRequest{
					Parent:     locationName,
					InstanceId: instanceID,
					Instance: &datafusionpb.Instance{
						DisplayName: "Team Instance",
						Description: "Stackyard Data Fusion instance",
						Type:        datafusionpb.Instance_BASIC,
					},
				})
				return err
			},
		},
		{
			name: "UpdateInstance",
			call: func(ctx context.Context, c *datafusion.Client) error {
				_, err := c.UpdateInstance(ctx, &datafusionpb.UpdateInstanceRequest{
					Instance: &datafusionpb.Instance{
						Name: instanceName,
						Labels: map[string]string{
							"env": "test",
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "RestartInstance",
			call: func(ctx context.Context, c *datafusion.Client) error {
				_, err := c.RestartInstance(ctx, &datafusionpb.RestartInstanceRequest{
					Name: instanceName,
				})
				return err
			},
		},
		{
			name: "DeleteInstance",
			call: func(ctx context.Context, c *datafusion.Client) error {
				_, err := c.DeleteInstance(ctx, &datafusionpb.DeleteInstanceRequest{
					Name: instanceName,
				})
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
		fmt.Fprintf(os.Stderr, "warning: close datafusion client: %v\n", err)
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
