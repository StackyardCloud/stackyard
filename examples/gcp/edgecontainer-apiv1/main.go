package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	edgecontainer "cloud.google.com/go/edgecontainer/apiv1"
	"cloud.google.com/go/edgecontainer/apiv1/edgecontainerpb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
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
	call func(context.Context, *edgecontainer.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	clusterID := getenv("STACKYARD_GCP_EDGECONTAINER_CLUSTER_ID", "team-cluster")
	nodePoolID := getenv("STACKYARD_GCP_EDGECONTAINER_NODE_POOL_ID", "default-pool")
	siteName := getenv("STACKYARD_GCP_EDGECONTAINER_SITE_NAME", fmt.Sprintf("projects/%s/locations/%s/zones/us-central1-a/sites/site-a", projectID, locationID))
	machineID := getenv("STACKYARD_GCP_EDGECONTAINER_MACHINE_ID", "machine-1")
	vpnConnectionID := getenv("STACKYARD_GCP_EDGECONTAINER_VPN_CONNECTION_ID", "vpn-1")
	operationID := getenv("STACKYARD_GCP_EDGECONTAINER_OPERATION_ID", "op-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	clusterName := locationName + "/clusters/" + clusterID
	nodePoolName := clusterName + "/nodePools/" + nodePoolID
	machineName := siteName + "/machines/" + machineID
	vpnConnectionName := locationName + "/vpnConnections/" + vpnConnectionID
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Distributed Cloud Edge Container apiv1 client using %s\n", apiEndpoint)

	client, err := edgecontainer.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create edgecontainer client: %v", err)
	}
	defer closeClient("edgecontainer client", client.Close)

	calls := []callSpec{
		{
			name: "ListClusters",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				it := c.ListClusters(ctx, &edgecontainerpb.ListClustersRequest{
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
			name: "GetCluster",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.GetCluster(ctx, &edgecontainerpb.GetClusterRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "CreateCluster",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.CreateCluster(ctx, &edgecontainerpb.CreateClusterRequest{
					Parent:    locationName,
					ClusterId: clusterID,
					Cluster: &edgecontainerpb.Cluster{
						Name: clusterName,
						Labels: map[string]string{
							"env": "local",
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateCluster",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.UpdateCluster(ctx, &edgecontainerpb.UpdateClusterRequest{
					Cluster: &edgecontainerpb.Cluster{
						Name: clusterName,
						Labels: map[string]string{
							"team": "platform",
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				})
				return err
			},
		},
		{
			name: "UpgradeCluster",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.UpgradeCluster(ctx, &edgecontainerpb.UpgradeClusterRequest{
					Name:          clusterName,
					TargetVersion: "1.29.0-gke.1",
					Schedule:      edgecontainerpb.UpgradeClusterRequest_IMMEDIATELY,
				})
				return err
			},
		},
		{
			name: "DeleteCluster",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.DeleteCluster(ctx, &edgecontainerpb.DeleteClusterRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "GenerateAccessToken",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.GenerateAccessToken(ctx, &edgecontainerpb.GenerateAccessTokenRequest{
					Cluster: clusterName,
				})
				return err
			},
		},
		{
			name: "GenerateOfflineCredential",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.GenerateOfflineCredential(ctx, &edgecontainerpb.GenerateOfflineCredentialRequest{
					Cluster: clusterName,
				})
				return err
			},
		},
		{
			name: "ListNodePools",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				it := c.ListNodePools(ctx, &edgecontainerpb.ListNodePoolsRequest{
					Parent:   clusterName,
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
			name: "GetNodePool",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.GetNodePool(ctx, &edgecontainerpb.GetNodePoolRequest{Name: nodePoolName})
				return err
			},
		},
		{
			name: "CreateNodePool",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.CreateNodePool(ctx, &edgecontainerpb.CreateNodePoolRequest{
					Parent:     clusterName,
					NodePoolId: nodePoolID,
					NodePool: &edgecontainerpb.NodePool{
						Name:      nodePoolName,
						NodeCount: 1,
					},
				})
				return err
			},
		},
		{
			name: "UpdateNodePool",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.UpdateNodePool(ctx, &edgecontainerpb.UpdateNodePoolRequest{
					NodePool: &edgecontainerpb.NodePool{
						Name: nodePoolName,
						Labels: map[string]string{
							"workload": "general",
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteNodePool",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.DeleteNodePool(ctx, &edgecontainerpb.DeleteNodePoolRequest{Name: nodePoolName})
				return err
			},
		},
		{
			name: "ListMachines",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				it := c.ListMachines(ctx, &edgecontainerpb.ListMachinesRequest{
					Parent:   siteName,
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
			name: "GetMachine",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.GetMachine(ctx, &edgecontainerpb.GetMachineRequest{Name: machineName})
				return err
			},
		},
		{
			name: "ListVpnConnections",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				it := c.ListVpnConnections(ctx, &edgecontainerpb.ListVpnConnectionsRequest{
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
			name: "GetVpnConnection",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.GetVpnConnection(ctx, &edgecontainerpb.GetVpnConnectionRequest{Name: vpnConnectionName})
				return err
			},
		},
		{
			name: "CreateVpnConnection",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.CreateVpnConnection(ctx, &edgecontainerpb.CreateVpnConnectionRequest{
					Parent:          locationName,
					VpnConnectionId: vpnConnectionID,
					VpnConnection: &edgecontainerpb.VpnConnection{
						Name:    vpnConnectionName,
						Cluster: clusterName,
						Vpc:     "projects/stackyard/global/networks/default",
					},
				})
				return err
			},
		},
		{
			name: "DeleteVpnConnection",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.DeleteVpnConnection(ctx, &edgecontainerpb.DeleteVpnConnectionRequest{Name: vpnConnectionName})
				return err
			},
		},
		{
			name: "GetServerConfig",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.GetServerConfig(ctx, &edgecontainerpb.GetServerConfigRequest{Name: locationName})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     "projects/" + projectID,
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
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
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
			call: func(ctx context.Context, c *edgecontainer.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *edgecontainer.Client) error {
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

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s: %v\n", label, err)
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
