package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type computeClients struct {
	zones            *compute.ZonesClient
	networks         *compute.NetworksClient
	instances        *compute.InstancesClient
	zoneOperations   *compute.ZoneOperationsClient
	globalOperations *compute.GlobalOperationsClient
}

type callSpec struct {
	name string
	call func(context.Context, *computeClients) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	zoneID := getenv("STACKYARD_GCP_COMPUTE_ZONE", "us-central1-a")
	instanceID := getenv("STACKYARD_GCP_COMPUTE_INSTANCE_ID", "team-vm")
	networkID := getenv("STACKYARD_GCP_COMPUTE_NETWORK_ID", "team-network")
	zoneOperationID := getenv("STACKYARD_GCP_COMPUTE_ZONE_OPERATION_ID", "op-zone-1")
	globalOperationID := getenv("STACKYARD_GCP_COMPUTE_GLOBAL_OPERATION_ID", "op-global-1")

	fmt.Printf("Stackyard GCP Compute Engine apiv1 client using %s\n", apiEndpoint)

	clients, err := createClients(ctx, apiEndpoint)
	if err != nil {
		exitf("failed to create compute clients: %v", err)
	}
	defer closeClients(clients)

	calls := []callSpec{
		{
			name: "ListZones",
			call: func(ctx context.Context, c *computeClients) error {
				it := c.zones.List(ctx, &computepb.ListZonesRequest{Project: projectID})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetZone",
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.zones.Get(ctx, &computepb.GetZoneRequest{
					Project: projectID,
					Zone:    zoneID,
				})
				return err
			},
		},
		{
			name: "ListNetworks",
			call: func(ctx context.Context, c *computeClients) error {
				it := c.networks.List(ctx, &computepb.ListNetworksRequest{Project: projectID})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetNetwork",
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.networks.Get(ctx, &computepb.GetNetworkRequest{
					Project: projectID,
					Network: networkID,
				})
				return err
			},
		},
		{
			name: "InsertNetwork",
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.networks.Insert(ctx, &computepb.InsertNetworkRequest{
					Project: projectID,
					NetworkResource: &computepb.Network{
						Name: stringPtr(networkID),
					},
				})
				return err
			},
		},
		{
			name: "ListInstances",
			call: func(ctx context.Context, c *computeClients) error {
				it := c.instances.List(ctx, &computepb.ListInstancesRequest{
					Project: projectID,
					Zone:    zoneID,
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
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.instances.Get(ctx, &computepb.GetInstanceRequest{
					Project:  projectID,
					Zone:     zoneID,
					Instance: instanceID,
				})
				return err
			},
		},
		{
			name: "InsertInstance",
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.instances.Insert(ctx, &computepb.InsertInstanceRequest{
					Project: projectID,
					Zone:    zoneID,
					InstanceResource: &computepb.Instance{
						Name: stringPtr(instanceID),
					},
				})
				return err
			},
		},
		{
			name: "StartInstance",
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.instances.Start(ctx, &computepb.StartInstanceRequest{
					Project:  projectID,
					Zone:     zoneID,
					Instance: instanceID,
				})
				return err
			},
		},
		{
			name: "StopInstance",
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.instances.Stop(ctx, &computepb.StopInstanceRequest{
					Project:  projectID,
					Zone:     zoneID,
					Instance: instanceID,
				})
				return err
			},
		},
		{
			name: "DeleteInstance",
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.instances.Delete(ctx, &computepb.DeleteInstanceRequest{
					Project:  projectID,
					Zone:     zoneID,
					Instance: instanceID,
				})
				return err
			},
		},
		{
			name: "ListZoneOperations",
			call: func(ctx context.Context, c *computeClients) error {
				it := c.zoneOperations.List(ctx, &computepb.ListZoneOperationsRequest{
					Project: projectID,
					Zone:    zoneID,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetZoneOperation",
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.zoneOperations.Get(ctx, &computepb.GetZoneOperationRequest{
					Project:   projectID,
					Zone:      zoneID,
					Operation: zoneOperationID,
				})
				return err
			},
		},
		{
			name: "WaitZoneOperation",
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.zoneOperations.Wait(ctx, &computepb.WaitZoneOperationRequest{
					Project:   projectID,
					Zone:      zoneID,
					Operation: zoneOperationID,
				})
				return err
			},
		},
		{
			name: "GetGlobalOperation",
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.globalOperations.Get(ctx, &computepb.GetGlobalOperationRequest{
					Project:   projectID,
					Operation: globalOperationID,
				})
				return err
			},
		},
		{
			name: "WaitGlobalOperation",
			call: func(ctx context.Context, c *computeClients) error {
				_, err := c.globalOperations.Wait(ctx, &computepb.WaitGlobalOperationRequest{
					Project:   projectID,
					Operation: globalOperationID,
				})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, clients)
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

func createClients(ctx context.Context, endpoint string) (*computeClients, error) {
	newClient := func() []option.ClientOption {
		return []option.ClientOption{
			option.WithEndpoint(endpoint),
			option.WithoutAuthentication(),
		}
	}

	zonesClient, err := compute.NewZonesRESTClient(ctx, newClient()...)
	if err != nil {
		return nil, err
	}

	networksClient, err := compute.NewNetworksRESTClient(ctx, newClient()...)
	if err != nil {
		closeNamedClient("zones", zonesClient.Close)
		return nil, err
	}

	instancesClient, err := compute.NewInstancesRESTClient(ctx, newClient()...)
	if err != nil {
		closeNamedClient("networks", networksClient.Close)
		closeNamedClient("zones", zonesClient.Close)
		return nil, err
	}

	zoneOperationsClient, err := compute.NewZoneOperationsRESTClient(ctx, newClient()...)
	if err != nil {
		closeNamedClient("instances", instancesClient.Close)
		closeNamedClient("networks", networksClient.Close)
		closeNamedClient("zones", zonesClient.Close)
		return nil, err
	}

	globalOperationsClient, err := compute.NewGlobalOperationsRESTClient(ctx, newClient()...)
	if err != nil {
		closeNamedClient("zone operations", zoneOperationsClient.Close)
		closeNamedClient("instances", instancesClient.Close)
		closeNamedClient("networks", networksClient.Close)
		closeNamedClient("zones", zonesClient.Close)
		return nil, err
	}

	return &computeClients{
		zones:            zonesClient,
		networks:         networksClient,
		instances:        instancesClient,
		zoneOperations:   zoneOperationsClient,
		globalOperations: globalOperationsClient,
	}, nil
}

func closeClients(clients *computeClients) {
	if clients == nil {
		return
	}
	closeNamedClient("global operations", clients.globalOperations.Close)
	closeNamedClient("zone operations", clients.zoneOperations.Close)
	closeNamedClient("instances", clients.instances.Close)
	closeNamedClient("networks", clients.networks.Close)
	closeNamedClient("zones", clients.zones.Close)
}

func closeNamedClient(name string, closeFn func() error) {
	if closeFn == nil {
		return
	}
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s compute client: %v\n", name, err)
	}
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

func stringPtr(v string) *string {
	return &v
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
