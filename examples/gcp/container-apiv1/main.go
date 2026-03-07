package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *container.ClusterManagerClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	clusterID := getenv("STACKYARD_GCP_CONTAINER_CLUSTER_ID", "team-cluster")
	nodePoolID := getenv("STACKYARD_GCP_CONTAINER_NODE_POOL_ID", "default-pool")
	operationID := getenv("STACKYARD_GCP_CONTAINER_OPERATION_ID", "op-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	clusterName := locationName + "/clusters/" + clusterID
	nodePoolName := clusterName + "/nodePools/" + nodePoolID
	operationName := locationName + "/operations/" + operationID
	projectName := "projects/" + projectID

	fmt.Printf("Stackyard GCP Kubernetes Engine apiv1 client using %s\n", apiEndpoint)

	client, err := container.NewClusterManagerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create container client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListClusters",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.ListClusters(ctx, &containerpb.ListClustersRequest{Parent: locationName})
				return err
			},
		},
		{
			name: "GetCluster",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.GetCluster(ctx, &containerpb.GetClusterRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "CreateCluster",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.CreateCluster(ctx, &containerpb.CreateClusterRequest{
					Parent:  locationName,
					Cluster: &containerpb.Cluster{Name: clusterID},
				})
				return err
			},
		},
		{
			name: "DeleteCluster",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.DeleteCluster(ctx, &containerpb.DeleteClusterRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "ListNodePools",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.ListNodePools(ctx, &containerpb.ListNodePoolsRequest{Parent: clusterName})
				return err
			},
		},
		{
			name: "GetNodePool",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.GetNodePool(ctx, &containerpb.GetNodePoolRequest{Name: nodePoolName})
				return err
			},
		},
		{
			name: "CreateNodePool",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.CreateNodePool(ctx, &containerpb.CreateNodePoolRequest{
					Parent:   clusterName,
					NodePool: &containerpb.NodePool{Name: nodePoolID},
				})
				return err
			},
		},
		{
			name: "DeleteNodePool",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.DeleteNodePool(ctx, &containerpb.DeleteNodePoolRequest{Name: nodePoolName})
				return err
			},
		},
		{
			name: "SetLoggingService",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.SetLoggingService(ctx, &containerpb.SetLoggingServiceRequest{
					Name:           clusterName,
					LoggingService: "logging.googleapis.com/kubernetes",
				})
				return err
			},
		},
		{
			name: "SetMonitoringService",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.SetMonitoringService(ctx, &containerpb.SetMonitoringServiceRequest{
					Name:              clusterName,
					MonitoringService: "monitoring.googleapis.com/kubernetes",
				})
				return err
			},
		},
		{
			name: "SetAddonsConfig",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.SetAddonsConfig(ctx, &containerpb.SetAddonsConfigRequest{
					Name:         clusterName,
					AddonsConfig: &containerpb.AddonsConfig{},
				})
				return err
			},
		},
		{
			name: "SetNodePoolManagement",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.SetNodePoolManagement(ctx, &containerpb.SetNodePoolManagementRequest{
					Name:       nodePoolName,
					Management: &containerpb.NodeManagement{AutoRepair: true, AutoUpgrade: true},
				})
				return err
			},
		},
		{
			name: "SetNodePoolSize",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.SetNodePoolSize(ctx, &containerpb.SetNodePoolSizeRequest{
					Name:      nodePoolName,
					NodeCount: 1,
				})
				return err
			},
		},
		{
			name: "CompleteNodePoolUpgrade",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				return c.CompleteNodePoolUpgrade(ctx, &containerpb.CompleteNodePoolUpgradeRequest{Name: nodePoolName})
			},
		},
		{
			name: "RollbackNodePoolUpgrade",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.RollbackNodePoolUpgrade(ctx, &containerpb.RollbackNodePoolUpgradeRequest{Name: nodePoolName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.ListOperations(ctx, &containerpb.ListOperationsRequest{Parent: locationName})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.GetOperation(ctx, &containerpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				return c.CancelOperation(ctx, &containerpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "GetServerConfig",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.GetServerConfig(ctx, &containerpb.GetServerConfigRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListUsableSubnetworks",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				it := c.ListUsableSubnetworks(ctx, &containerpb.ListUsableSubnetworksRequest{
					Parent:   projectName,
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
			name: "CheckAutopilotCompatibility",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.CheckAutopilotCompatibility(ctx, &containerpb.CheckAutopilotCompatibilityRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "FetchClusterUpgradeInfo",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.FetchClusterUpgradeInfo(ctx, &containerpb.FetchClusterUpgradeInfoRequest{
					Name:    clusterName,
					Version: "v1",
				})
				return err
			},
		},
		{
			name: "FetchNodePoolUpgradeInfo",
			call: func(ctx context.Context, c *container.ClusterManagerClient) error {
				_, err := c.FetchNodePoolUpgradeInfo(ctx, &containerpb.FetchNodePoolUpgradeInfoRequest{
					Name:    nodePoolName,
					Version: "v1",
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
		fmt.Fprintf(os.Stderr, "warning: close container client: %v\n", err)
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
