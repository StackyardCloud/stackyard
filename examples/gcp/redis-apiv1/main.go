package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	redis "cloud.google.com/go/redis/apiv1"
	"cloud.google.com/go/redis/apiv1/redispb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type callSpec struct {
	name string
	call func(context.Context, *redis.CloudRedisClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	instanceID := getenv("STACKYARD_GCP_REDIS_INSTANCE_ID", "redis-1")

	projectParent := fmt.Sprintf("projects/%s", projectID)
	locationParent := fmt.Sprintf("%s/locations/%s", projectParent, locationID)
	instanceName := fmt.Sprintf("%s/instances/%s", locationParent, instanceID)
	operationName := fmt.Sprintf("%s/operations/op-1", locationParent)

	fmt.Printf("Stackyard GCP Redis apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "redis",
		},
	}

	client, err := redis.NewCloudRedisRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create redis client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{Name: projectParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationParent})
				return err
			},
		},
		{
			name: "ListInstances",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				it := c.ListInstances(ctx, &redispb.ListInstancesRequest{Parent: locationParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetInstance",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				_, err := c.GetInstance(ctx, &redispb.GetInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "GetInstanceAuthString",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				_, err := c.GetInstanceAuthString(ctx, &redispb.GetInstanceAuthStringRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "CreateInstance",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				op, err := c.CreateInstance(ctx, &redispb.CreateInstanceRequest{
					Parent:     locationParent,
					InstanceId: instanceID,
					Instance: &redispb.Instance{
						Name:         instanceName,
						DisplayName:  "Stackyard Redis",
						Tier:         redispb.Instance_STANDARD_HA,
						MemorySizeGb: 4,
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "UpdateInstance",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				op, err := c.UpdateInstance(ctx, &redispb.UpdateInstanceRequest{
					Instance: &redispb.Instance{
						Name:        instanceName,
						DisplayName: "Stackyard Redis Updated",
						Labels:      map[string]string{"env": "test"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "labels"}},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "UpgradeInstance",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				op, err := c.UpgradeInstance(ctx, &redispb.UpgradeInstanceRequest{
					Name:         instanceName,
					RedisVersion: "REDIS_7_0",
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "ImportInstance",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				op, err := c.ImportInstance(ctx, &redispb.ImportInstanceRequest{
					Name: instanceName,
					InputConfig: &redispb.InputConfig{
						Source: &redispb.InputConfig_GcsSource{
							GcsSource: &redispb.GcsSource{Uri: "gs://stackyard/import.rdb"},
						},
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "ExportInstance",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				op, err := c.ExportInstance(ctx, &redispb.ExportInstanceRequest{
					Name: instanceName,
					OutputConfig: &redispb.OutputConfig{
						Destination: &redispb.OutputConfig_GcsDestination{
							GcsDestination: &redispb.GcsDestination{Uri: "gs://stackyard/export.rdb"},
						},
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "FailoverInstance",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				op, err := c.FailoverInstance(ctx, &redispb.FailoverInstanceRequest{
					Name:               instanceName,
					DataProtectionMode: redispb.FailoverInstanceRequest_LIMITED_DATA_LOSS,
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "RescheduleMaintenance",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				op, err := c.RescheduleMaintenance(ctx, &redispb.RescheduleMaintenanceRequest{
					Name:           instanceName,
					RescheduleType: redispb.RescheduleMaintenanceRequest_SPECIFIC_TIME,
					ScheduleTime:   timestamppb.New(time.Now().UTC().Add(2 * time.Hour)),
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "DeleteInstance",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				op, err := c.DeleteInstance(ctx, &redispb.DeleteInstanceRequest{Name: instanceName})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: locationParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *redis.CloudRedisClient) error {
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

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close redis client: %v\n", err)
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

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	return base.RoundTrip(clone)
}
