package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	netapp "cloud.google.com/go/netapp/apiv1"
	"cloud.google.com/go/netapp/apiv1/netapppb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *netapp.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	storagePoolID := getenv("STACKYARD_GCP_NETAPP_STORAGE_POOL_ID", "pool-a")
	volumeID := getenv("STACKYARD_GCP_NETAPP_VOLUME_ID", "vol-a")
	backupVaultID := getenv("STACKYARD_GCP_NETAPP_BACKUP_VAULT_ID", "vault-a")

	locationParent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	storagePoolName := fmt.Sprintf("%s/storagePools/%s", locationParent, storagePoolID)
	volumeName := fmt.Sprintf("%s/volumes/%s", locationParent, volumeID)
	backupVaultName := fmt.Sprintf("%s/backupVaults/%s", locationParent, backupVaultID)

	fmt.Printf("Stackyard GCP NetApp apiv1 client using %s\n", apiEndpoint)

	client, err := netapp.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create netapp client: %v", err)
	}
	defer closeClient("netapp", client.Close)

	calls := []callSpec{
		{
			name: "ListStoragePools",
			call: func(ctx context.Context, c *netapp.Client) error {
				it := c.ListStoragePools(ctx, &netapppb.ListStoragePoolsRequest{
					Parent:   locationParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetStoragePool",
			call: func(ctx context.Context, c *netapp.Client) error {
				_, err := c.GetStoragePool(ctx, &netapppb.GetStoragePoolRequest{
					Name: storagePoolName,
				})
				return err
			},
		},
		{
			name: "ListVolumes",
			call: func(ctx context.Context, c *netapp.Client) error {
				it := c.ListVolumes(ctx, &netapppb.ListVolumesRequest{
					Parent:   locationParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetVolume",
			call: func(ctx context.Context, c *netapp.Client) error {
				_, err := c.GetVolume(ctx, &netapppb.GetVolumeRequest{
					Name: volumeName,
				})
				return err
			},
		},
		{
			name: "CreateVolume",
			call: func(ctx context.Context, c *netapp.Client) error {
				_, err := c.CreateVolume(ctx, &netapppb.CreateVolumeRequest{
					Parent:   locationParent,
					VolumeId: volumeID,
					Volume: &netapppb.Volume{
						ShareName:   "share-a",
						StoragePool: storagePoolName,
						CapacityGib: 100,
						Protocols:   []netapppb.Protocols{netapppb.Protocols_NFSV3},
					},
				})
				return err
			},
		},
		{
			name: "DeleteVolume",
			call: func(ctx context.Context, c *netapp.Client) error {
				_, err := c.DeleteVolume(ctx, &netapppb.DeleteVolumeRequest{
					Name: volumeName,
				})
				return err
			},
		},
		{
			name: "ListActiveDirectories",
			call: func(ctx context.Context, c *netapp.Client) error {
				it := c.ListActiveDirectories(ctx, &netapppb.ListActiveDirectoriesRequest{
					Parent:   locationParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListBackupVaults",
			call: func(ctx context.Context, c *netapp.Client) error {
				it := c.ListBackupVaults(ctx, &netapppb.ListBackupVaultsRequest{
					Parent:   locationParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetBackupVault",
			call: func(ctx context.Context, c *netapp.Client) error {
				_, err := c.GetBackupVault(ctx, &netapppb.GetBackupVaultRequest{
					Name: backupVaultName,
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

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
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
