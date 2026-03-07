package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	filestore "cloud.google.com/go/filestore/apiv1"
	"cloud.google.com/go/filestore/apiv1/filestorepb"
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
	call func(context.Context, *filestore.CloudFilestoreManagerClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	zone := getenv("STACKYARD_GCP_FILESTORE_ZONE", "us-central1-c")
	region := getenv("STACKYARD_GCP_FILESTORE_REGION", "us-central1")
	instanceID := getenv("STACKYARD_GCP_FILESTORE_INSTANCE_ID", "team-instance")
	snapshotID := getenv("STACKYARD_GCP_FILESTORE_SNAPSHOT_ID", "team-snapshot")
	backupID := getenv("STACKYARD_GCP_FILESTORE_BACKUP_ID", "team-backup")
	operationID := getenv("STACKYARD_GCP_FILESTORE_OPERATION_ID", "op-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	zoneParent := projectName + "/locations/" + zone
	regionParent := projectName + "/locations/" + region
	instanceName := zoneParent + "/instances/" + instanceID
	snapshotName := instanceName + "/snapshots/" + snapshotID
	backupName := regionParent + "/backups/" + backupID
	operationName := zoneParent + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Cloud Filestore apiv1 client using %s\n", apiEndpoint)

	client, err := filestore.NewCloudFilestoreManagerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloud filestore manager client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListInstances",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				it := c.ListInstances(ctx, &filestorepb.ListInstancesRequest{
					Parent:   zoneParent,
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
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.GetInstance(ctx, &filestorepb.GetInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "CreateInstance",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.CreateInstance(ctx, &filestorepb.CreateInstanceRequest{
					Parent:     zoneParent,
					InstanceId: instanceID,
					Instance:   &filestorepb.Instance{Name: instanceName},
				})
				return err
			},
		},
		{
			name: "UpdateInstance",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.UpdateInstance(ctx, &filestorepb.UpdateInstanceRequest{
					Instance:   &filestorepb.Instance{Name: instanceName, Description: "updated by stackyard example"},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "RestoreInstance",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.RestoreInstance(ctx, &filestorepb.RestoreInstanceRequest{
					Name:      instanceName,
					FileShare: "vol1",
					Source:    &filestorepb.RestoreInstanceRequest_SourceBackup{SourceBackup: backupName},
				})
				return err
			},
		},
		{
			name: "RevertInstance",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.RevertInstance(ctx, &filestorepb.RevertInstanceRequest{
					Name:             instanceName,
					TargetSnapshotId: snapshotID,
				})
				return err
			},
		},
		{
			name: "DeleteInstance",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.DeleteInstance(ctx, &filestorepb.DeleteInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "ListSnapshots",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				it := c.ListSnapshots(ctx, &filestorepb.ListSnapshotsRequest{
					Parent:   instanceName,
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
			name: "GetSnapshot",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.GetSnapshot(ctx, &filestorepb.GetSnapshotRequest{Name: snapshotName})
				return err
			},
		},
		{
			name: "CreateSnapshot",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.CreateSnapshot(ctx, &filestorepb.CreateSnapshotRequest{
					Parent:     instanceName,
					SnapshotId: snapshotID,
					Snapshot:   &filestorepb.Snapshot{Name: snapshotName},
				})
				return err
			},
		},
		{
			name: "UpdateSnapshot",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.UpdateSnapshot(ctx, &filestorepb.UpdateSnapshotRequest{
					Snapshot:   &filestorepb.Snapshot{Name: snapshotName, Description: "updated by stackyard example"},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "DeleteSnapshot",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.DeleteSnapshot(ctx, &filestorepb.DeleteSnapshotRequest{Name: snapshotName})
				return err
			},
		},
		{
			name: "ListBackups",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				it := c.ListBackups(ctx, &filestorepb.ListBackupsRequest{
					Parent:   regionParent,
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
			name: "GetBackup",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.GetBackup(ctx, &filestorepb.GetBackupRequest{Name: backupName})
				return err
			},
		},
		{
			name: "CreateBackup",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.CreateBackup(ctx, &filestorepb.CreateBackupRequest{
					Parent:   regionParent,
					BackupId: backupID,
					Backup:   &filestorepb.Backup{Name: backupName},
				})
				return err
			},
		},
		{
			name: "UpdateBackup",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.UpdateBackup(ctx, &filestorepb.UpdateBackupRequest{
					Backup:     &filestorepb.Backup{Name: backupName, Description: "updated by stackyard example"},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "DeleteBackup",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.DeleteBackup(ctx, &filestorepb.DeleteBackupRequest{Name: backupName})
				return err
			},
		},
		{
			name: "PromoteReplica",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.PromoteReplica(ctx, &filestorepb.PromoteReplicaRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: zoneParent})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
					Filter:   "locationId:*",
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
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     zoneParent,
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
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *filestore.CloudFilestoreManagerClient) error {
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
		fmt.Fprintf(os.Stderr, "warning: close cloud filestore manager client: %v\n", err)
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
