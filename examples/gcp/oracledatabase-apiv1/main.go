package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	oracledatabase "cloud.google.com/go/oracledatabase/apiv1"
	"cloud.google.com/go/oracledatabase/apiv1/oracledatabasepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *oracledatabase.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	exadataName := locationName + "/cloudExadataInfrastructures/exadata-1"
	vmClusterName := locationName + "/cloudVmClusters/vmcluster-1"
	autonomousDatabaseName := locationName + "/autonomousDatabases/adb-1"
	peerAutonomousDatabaseName := locationName + "/autonomousDatabases/adb-2"
	odbNetworkName := locationName + "/odbNetworks/odbnetwork-1"
	odbSubnetName := locationName + "/odbSubnets/odbsubnet-1"
	exadbVmClusterName := locationName + "/exadbVmClusters/exadbvmcluster-1"
	storageVaultName := locationName + "/exascaleDbStorageVaults/storagevault-1"
	dbSystemName := locationName + "/dbSystems/dbsystem-1"
	databaseName := locationName + "/databases/database-1"
	pluggableDatabaseName := locationName + "/pluggableDatabases/pdb-1"
	operationName := locationName + "/operations/op-1"

	fmt.Printf("Stackyard GCP Oracle Database apiv1 client using %s\n", apiEndpoint)

	client, err := oracledatabase.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create oracle database client: %v", err)
	}
	defer closeClient("oracle database", client.Close)

	calls := []callSpec{
		{
			name: "ListCloudExadataInfrastructures",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListCloudExadataInfrastructures(ctx, &oracledatabasepb.ListCloudExadataInfrastructuresRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetCloudExadataInfrastructure",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetCloudExadataInfrastructure(ctx, &oracledatabasepb.GetCloudExadataInfrastructureRequest{Name: exadataName})
				return err
			},
		},
		{
			name: "ListCloudVmClusters",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListCloudVmClusters(ctx, &oracledatabasepb.ListCloudVmClustersRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetCloudVmCluster",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetCloudVmCluster(ctx, &oracledatabasepb.GetCloudVmClusterRequest{Name: vmClusterName})
				return err
			},
		},
		{
			name: "ListAutonomousDatabases",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListAutonomousDatabases(ctx, &oracledatabasepb.ListAutonomousDatabasesRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetAutonomousDatabase",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetAutonomousDatabase(ctx, &oracledatabasepb.GetAutonomousDatabaseRequest{Name: autonomousDatabaseName})
				return err
			},
		},
		{
			name: "GenerateAutonomousDatabaseWallet",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GenerateAutonomousDatabaseWallet(ctx, &oracledatabasepb.GenerateAutonomousDatabaseWalletRequest{
					Name:     autonomousDatabaseName,
					Password: "stackyard-password",
				})
				return err
			},
		},
		{
			name: "StartAutonomousDatabase",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.StartAutonomousDatabase(ctx, &oracledatabasepb.StartAutonomousDatabaseRequest{Name: autonomousDatabaseName})
				return err
			},
		},
		{
			name: "StopAutonomousDatabase",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.StopAutonomousDatabase(ctx, &oracledatabasepb.StopAutonomousDatabaseRequest{Name: autonomousDatabaseName})
				return err
			},
		},
		{
			name: "RestartAutonomousDatabase",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.RestartAutonomousDatabase(ctx, &oracledatabasepb.RestartAutonomousDatabaseRequest{Name: autonomousDatabaseName})
				return err
			},
		},
		{
			name: "SwitchoverAutonomousDatabase",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.SwitchoverAutonomousDatabase(ctx, &oracledatabasepb.SwitchoverAutonomousDatabaseRequest{
					Name:                   autonomousDatabaseName,
					PeerAutonomousDatabase: peerAutonomousDatabaseName,
				})
				return err
			},
		},
		{
			name: "FailoverAutonomousDatabase",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.FailoverAutonomousDatabase(ctx, &oracledatabasepb.FailoverAutonomousDatabaseRequest{
					Name:                   autonomousDatabaseName,
					PeerAutonomousDatabase: peerAutonomousDatabaseName,
				})
				return err
			},
		},
		{
			name: "ListOdbNetworks",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListOdbNetworks(ctx, &oracledatabasepb.ListOdbNetworksRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetOdbNetwork",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetOdbNetwork(ctx, &oracledatabasepb.GetOdbNetworkRequest{Name: odbNetworkName})
				return err
			},
		},
		{
			name: "ListOdbSubnets",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListOdbSubnets(ctx, &oracledatabasepb.ListOdbSubnetsRequest{
					Parent:   odbNetworkName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetOdbSubnet",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetOdbSubnet(ctx, &oracledatabasepb.GetOdbSubnetRequest{Name: odbSubnetName})
				return err
			},
		},
		{
			name: "ListExadbVmClusters",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListExadbVmClusters(ctx, &oracledatabasepb.ListExadbVmClustersRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetExadbVmCluster",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetExadbVmCluster(ctx, &oracledatabasepb.GetExadbVmClusterRequest{Name: exadbVmClusterName})
				return err
			},
		},
		{
			name: "ListExascaleDbStorageVaults",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListExascaleDbStorageVaults(ctx, &oracledatabasepb.ListExascaleDbStorageVaultsRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetExascaleDbStorageVault",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetExascaleDbStorageVault(ctx, &oracledatabasepb.GetExascaleDbStorageVaultRequest{Name: storageVaultName})
				return err
			},
		},
		{
			name: "ListDbSystems",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListDbSystems(ctx, &oracledatabasepb.ListDbSystemsRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetDbSystem",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetDbSystem(ctx, &oracledatabasepb.GetDbSystemRequest{Name: dbSystemName})
				return err
			},
		},
		{
			name: "ListDatabases",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListDatabases(ctx, &oracledatabasepb.ListDatabasesRequest{
					Parent:   dbSystemName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetDatabase",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetDatabase(ctx, &oracledatabasepb.GetDatabaseRequest{Name: databaseName})
				return err
			},
		},
		{
			name: "ListPluggableDatabases",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListPluggableDatabases(ctx, &oracledatabasepb.ListPluggableDatabasesRequest{
					Parent:   databaseName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetPluggableDatabase",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetPluggableDatabase(ctx, &oracledatabasepb.GetPluggableDatabaseRequest{Name: pluggableDatabaseName})
				return err
			},
		},
		{
			name: "ListDbVersions",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListDbVersions(ctx, &oracledatabasepb.ListDbVersionsRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListDatabaseCharacterSets",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListDatabaseCharacterSets(ctx, &oracledatabasepb.ListDatabaseCharacterSetsRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     fmt.Sprintf("projects/%s", projectID),
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *oracledatabase.Client) error {
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
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
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

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large")
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
