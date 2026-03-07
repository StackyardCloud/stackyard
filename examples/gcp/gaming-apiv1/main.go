package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	gaming "cloud.google.com/go/gaming/apiv1"
	"cloud.google.com/go/gaming/apiv1/gamingpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type sdkClients struct {
	realms      *gaming.RealmsClient
	clusters    *gaming.GameServerClustersClient
	configs     *gaming.GameServerConfigsClient
	deployments *gaming.GameServerDeploymentsClient
}

type callSpec struct {
	name string
	call func(context.Context, *sdkClients) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "global")
	realmID := getenv("STACKYARD_GCP_GAMING_REALM_ID", "team-realm")
	clusterID := getenv("STACKYARD_GCP_GAMING_CLUSTER_ID", "team-cluster")
	deploymentID := getenv("STACKYARD_GCP_GAMING_DEPLOYMENT_ID", "team-deployment")
	configID := getenv("STACKYARD_GCP_GAMING_CONFIG_ID", "config-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	realmName := locationName + "/realms/" + realmID
	clusterName := realmName + "/gameServerClusters/" + clusterID
	deploymentName := locationName + "/gameServerDeployments/" + deploymentID
	configName := deploymentName + "/configs/" + configID
	rolloutName := deploymentName + "/rollout"

	fmt.Printf("Stackyard GCP Game Services apiv1 client using %s\n", apiEndpoint)

	clients, err := newSDKClients(ctx, apiEndpoint)
	if err != nil {
		exitf("failed to create gaming clients: %v", err)
	}
	defer closeClient("realms", clients.realms.Close)
	defer closeClient("clusters", clients.clusters.Close)
	defer closeClient("configs", clients.configs.Close)
	defer closeClient("deployments", clients.deployments.Close)

	calls := []callSpec{
		{
			name: "ListRealms",
			call: func(ctx context.Context, c *sdkClients) error {
				it := c.realms.ListRealms(ctx, &gamingpb.ListRealmsRequest{
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
			name: "GetRealm",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.realms.GetRealm(ctx, &gamingpb.GetRealmRequest{Name: realmName})
				return err
			},
		},
		{
			name: "CreateRealm",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.realms.CreateRealm(ctx, &gamingpb.CreateRealmRequest{
					Parent:  locationName,
					RealmId: realmID,
					Realm:   sampleRealm(realmName),
				})
				return err
			},
		},
		{
			name: "PreviewRealmUpdate",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.realms.PreviewRealmUpdate(ctx, &gamingpb.PreviewRealmUpdateRequest{
					Realm: &gamingpb.Realm{
						Name:        realmName,
						Description: "preview update",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "UpdateRealm",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.realms.UpdateRealm(ctx, &gamingpb.UpdateRealmRequest{
					Realm: &gamingpb.Realm{
						Name:        realmName,
						Description: "updated by stackyard example",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "DeleteRealm",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.realms.DeleteRealm(ctx, &gamingpb.DeleteRealmRequest{Name: realmName})
				return err
			},
		},
		{
			name: "ListGameServerClusters",
			call: func(ctx context.Context, c *sdkClients) error {
				it := c.clusters.ListGameServerClusters(ctx, &gamingpb.ListGameServerClustersRequest{
					Parent:   realmName,
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
			name: "GetGameServerCluster",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.clusters.GetGameServerCluster(ctx, &gamingpb.GetGameServerClusterRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "CreateGameServerCluster",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.clusters.CreateGameServerCluster(ctx, &gamingpb.CreateGameServerClusterRequest{
					Parent:              realmName,
					GameServerClusterId: clusterID,
					GameServerCluster:   sampleCluster(clusterName),
				})
				return err
			},
		},
		{
			name: "PreviewCreateGameServerCluster",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.clusters.PreviewCreateGameServerCluster(ctx, &gamingpb.PreviewCreateGameServerClusterRequest{
					Parent:              realmName,
					GameServerClusterId: clusterID,
					GameServerCluster:   sampleCluster(clusterName),
				})
				return err
			},
		},
		{
			name: "PreviewUpdateGameServerCluster",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.clusters.PreviewUpdateGameServerCluster(ctx, &gamingpb.PreviewUpdateGameServerClusterRequest{
					GameServerCluster: &gamingpb.GameServerCluster{
						Name:        clusterName,
						Description: "preview update",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "PreviewDeleteGameServerCluster",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.clusters.PreviewDeleteGameServerCluster(ctx, &gamingpb.PreviewDeleteGameServerClusterRequest{
					Name: clusterName,
				})
				return err
			},
		},
		{
			name: "UpdateGameServerCluster",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.clusters.UpdateGameServerCluster(ctx, &gamingpb.UpdateGameServerClusterRequest{
					GameServerCluster: &gamingpb.GameServerCluster{
						Name:        clusterName,
						Description: "updated by stackyard example",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "DeleteGameServerCluster",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.clusters.DeleteGameServerCluster(ctx, &gamingpb.DeleteGameServerClusterRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "ListGameServerDeployments",
			call: func(ctx context.Context, c *sdkClients) error {
				it := c.deployments.ListGameServerDeployments(ctx, &gamingpb.ListGameServerDeploymentsRequest{
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
			name: "GetGameServerDeployment",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.deployments.GetGameServerDeployment(ctx, &gamingpb.GetGameServerDeploymentRequest{
					Name: deploymentName,
				})
				return err
			},
		},
		{
			name: "CreateGameServerDeployment",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.deployments.CreateGameServerDeployment(ctx, &gamingpb.CreateGameServerDeploymentRequest{
					Parent:               locationName,
					DeploymentId:         deploymentID,
					GameServerDeployment: sampleDeployment(deploymentName),
				})
				return err
			},
		},
		{
			name: "UpdateGameServerDeployment",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.deployments.UpdateGameServerDeployment(ctx, &gamingpb.UpdateGameServerDeploymentRequest{
					GameServerDeployment: &gamingpb.GameServerDeployment{
						Name:   deploymentName,
						Labels: map[string]string{"owner": "stackyard"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "DeleteGameServerDeployment",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.deployments.DeleteGameServerDeployment(ctx, &gamingpb.DeleteGameServerDeploymentRequest{
					Name: deploymentName,
				})
				return err
			},
		},
		{
			name: "GetGameServerDeploymentRollout",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.deployments.GetGameServerDeploymentRollout(ctx, &gamingpb.GetGameServerDeploymentRolloutRequest{
					Name: rolloutName,
				})
				return err
			},
		},
		{
			name: "PreviewGameServerDeploymentRollout",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.deployments.PreviewGameServerDeploymentRollout(ctx, &gamingpb.PreviewGameServerDeploymentRolloutRequest{
					Rollout: &gamingpb.GameServerDeploymentRollout{
						Name:                    rolloutName,
						DefaultGameServerConfig: configName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"default_game_server_config"}},
				})
				return err
			},
		},
		{
			name: "UpdateGameServerDeploymentRollout",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.deployments.UpdateGameServerDeploymentRollout(ctx, &gamingpb.UpdateGameServerDeploymentRolloutRequest{
					Rollout: &gamingpb.GameServerDeploymentRollout{
						Name:                    rolloutName,
						DefaultGameServerConfig: configName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"default_game_server_config"}},
				})
				return err
			},
		},
		{
			name: "FetchDeploymentState",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.deployments.FetchDeploymentState(ctx, &gamingpb.FetchDeploymentStateRequest{
					Name: deploymentName,
				})
				return err
			},
		},
		{
			name: "ListGameServerConfigs",
			call: func(ctx context.Context, c *sdkClients) error {
				it := c.configs.ListGameServerConfigs(ctx, &gamingpb.ListGameServerConfigsRequest{
					Parent:   deploymentName,
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
			name: "GetGameServerConfig",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.configs.GetGameServerConfig(ctx, &gamingpb.GetGameServerConfigRequest{
					Name: configName,
				})
				return err
			},
		},
		{
			name: "CreateGameServerConfig",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.configs.CreateGameServerConfig(ctx, &gamingpb.CreateGameServerConfigRequest{
					Parent:           deploymentName,
					ConfigId:         configID,
					GameServerConfig: sampleConfig(configName),
				})
				return err
			},
		},
		{
			name: "DeleteGameServerConfig",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.configs.DeleteGameServerConfig(ctx, &gamingpb.DeleteGameServerConfigRequest{
					Name: configName,
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

func newSDKClients(ctx context.Context, endpoint string) (*sdkClients, error) {
	realmsClient, err := gaming.NewRealmsRESTClient(ctx,
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		return nil, err
	}

	clustersClient, err := gaming.NewGameServerClustersRESTClient(ctx,
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		closeClient("realms", realmsClient.Close)
		return nil, err
	}

	configsClient, err := gaming.NewGameServerConfigsRESTClient(ctx,
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		closeClient("clusters", clustersClient.Close)
		closeClient("realms", realmsClient.Close)
		return nil, err
	}

	deploymentsClient, err := gaming.NewGameServerDeploymentsRESTClient(ctx,
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		closeClient("configs", configsClient.Close)
		closeClient("clusters", clustersClient.Close)
		closeClient("realms", realmsClient.Close)
		return nil, err
	}

	return &sdkClients{
		realms:      realmsClient,
		clusters:    clustersClient,
		configs:     configsClient,
		deployments: deploymentsClient,
	}, nil
}

func sampleRealm(name string) *gamingpb.Realm {
	return &gamingpb.Realm{
		Name:        name,
		TimeZone:    "UTC",
		Description: "stackyard realm",
	}
}

func sampleCluster(name string) *gamingpb.GameServerCluster {
	return &gamingpb.GameServerCluster{
		Name:        name,
		Description: "stackyard game server cluster",
		ConnectionInfo: &gamingpb.GameServerClusterConnectionInfo{
			ClusterReference: &gamingpb.GameServerClusterConnectionInfo_GkeClusterReference{
				GkeClusterReference: &gamingpb.GkeClusterReference{
					Cluster: "projects/stackyard/locations/us-central1/clusters/dev",
				},
			},
			Namespace: "default",
		},
	}
}

func sampleDeployment(name string) *gamingpb.GameServerDeployment {
	return &gamingpb.GameServerDeployment{
		Name:        name,
		Description: "stackyard deployment",
	}
}

func sampleConfig(name string) *gamingpb.GameServerConfig {
	return &gamingpb.GameServerConfig{
		Name:        name,
		Description: "stackyard game server config",
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
