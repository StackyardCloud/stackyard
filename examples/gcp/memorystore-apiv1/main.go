package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	memorystore "cloud.google.com/go/memorystore/apiv1"
	"cloud.google.com/go/memorystore/apiv1/memorystorepb"
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
	call func(context.Context, *memorystore.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	instanceID := getenv("STACKYARD_GCP_MEMORYSTORE_INSTANCE_ID", "redis-a")
	backupCollectionID := getenv("STACKYARD_GCP_MEMORYSTORE_BACKUP_COLLECTION_ID", "default")
	backupID := getenv("STACKYARD_GCP_MEMORYSTORE_BACKUP_ID", "backup-1")
	operationID := getenv("STACKYARD_GCP_MEMORYSTORE_OPERATION_ID", "op-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	parent := projectName + "/locations/" + locationID
	instanceName := parent + "/instances/" + instanceID
	certificateAuthorityName := instanceName + "/certificateAuthority"
	backupCollectionName := parent + "/backupCollections/" + backupCollectionID
	backupName := backupCollectionName + "/backups/" + backupID
	operationName := parent + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Memorystore apiv1 client using %s\n", apiEndpoint)

	client, err := memorystore.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create memorystore client: %v", err)
	}
	defer closeClient("memorystore", client.Close)

	calls := []callSpec{
		{
			name: "ListInstances",
			call: func(ctx context.Context, c *memorystore.Client) error {
				it := c.ListInstances(ctx, &memorystorepb.ListInstancesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetInstance",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.GetInstance(ctx, &memorystorepb.GetInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "CreateInstance",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.CreateInstance(ctx, &memorystorepb.CreateInstanceRequest{
					Parent:     parent,
					InstanceId: instanceID,
					Instance: &memorystorepb.Instance{
						Name:       instanceName,
						ShardCount: 1,
					},
				})
				return err
			},
		},
		{
			name: "UpdateInstance",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.UpdateInstance(ctx, &memorystorepb.UpdateInstanceRequest{
					Instance: &memorystorepb.Instance{
						Name:   instanceName,
						Labels: map[string]string{"owner": "stackyard"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "DeleteInstance",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.DeleteInstance(ctx, &memorystorepb.DeleteInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "GetCertificateAuthority",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.GetCertificateAuthority(ctx, &memorystorepb.GetCertificateAuthorityRequest{Name: certificateAuthorityName})
				return err
			},
		},
		{
			name: "RescheduleMaintenance",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.RescheduleMaintenance(ctx, &memorystorepb.RescheduleMaintenanceRequest{
					Name:           instanceName,
					RescheduleType: memorystorepb.RescheduleMaintenanceRequest_IMMEDIATE,
				})
				return err
			},
		},
		{
			name: "ListBackupCollections",
			call: func(ctx context.Context, c *memorystore.Client) error {
				it := c.ListBackupCollections(ctx, &memorystorepb.ListBackupCollectionsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetBackupCollection",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.GetBackupCollection(ctx, &memorystorepb.GetBackupCollectionRequest{Name: backupCollectionName})
				return err
			},
		},
		{
			name: "ListBackups",
			call: func(ctx context.Context, c *memorystore.Client) error {
				it := c.ListBackups(ctx, &memorystorepb.ListBackupsRequest{
					Parent:   backupCollectionName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetBackup",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.GetBackup(ctx, &memorystorepb.GetBackupRequest{Name: backupName})
				return err
			},
		},
		{
			name: "DeleteBackup",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.DeleteBackup(ctx, &memorystorepb.DeleteBackupRequest{Name: backupName})
				return err
			},
		},
		{
			name: "ExportBackup",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.ExportBackup(ctx, &memorystorepb.ExportBackupRequest{
					Name: backupName,
					Destination: &memorystorepb.ExportBackupRequest_GcsBucket{
						GcsBucket: "gs://stackyard-memorystore-export",
					},
				})
				return err
			},
		},
		{
			name: "BackupInstance",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.BackupInstance(ctx, &memorystorepb.BackupInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *memorystore.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *memorystore.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *memorystore.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *memorystore.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *memorystore.Client) error {
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
