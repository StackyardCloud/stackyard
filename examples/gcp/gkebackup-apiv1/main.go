package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	gkebackup "cloud.google.com/go/gkebackup/apiv1"
	"cloud.google.com/go/gkebackup/apiv1/gkebackuppb"
	iampb "cloud.google.com/go/iam/apiv1/iampb"
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
	call func(context.Context, *gkebackup.BackupForGKEClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	clusterID := getenv("STACKYARD_GCP_GKEBACKUP_CLUSTER_ID", "primary")
	backupPlanID := getenv("STACKYARD_GCP_GKEBACKUP_BACKUP_PLAN_ID", "plan-a")
	backupChannelID := getenv("STACKYARD_GCP_GKEBACKUP_BACKUP_CHANNEL_ID", "channel-a")
	backupBindingID := getenv("STACKYARD_GCP_GKEBACKUP_BACKUP_BINDING_ID", "binding-a")
	backupID := getenv("STACKYARD_GCP_GKEBACKUP_BACKUP_ID", "backup-a")
	volumeBackupID := getenv("STACKYARD_GCP_GKEBACKUP_VOLUME_BACKUP_ID", "volume-backup-a")
	restorePlanID := getenv("STACKYARD_GCP_GKEBACKUP_RESTORE_PLAN_ID", "restore-plan-a")
	restoreChannelID := getenv("STACKYARD_GCP_GKEBACKUP_RESTORE_CHANNEL_ID", "restore-channel-a")
	restoreBindingID := getenv("STACKYARD_GCP_GKEBACKUP_RESTORE_BINDING_ID", "restore-binding-a")
	restoreID := getenv("STACKYARD_GCP_GKEBACKUP_RESTORE_ID", "restore-a")
	volumeRestoreID := getenv("STACKYARD_GCP_GKEBACKUP_VOLUME_RESTORE_ID", "volume-restore-a")
	operationID := getenv("STACKYARD_GCP_GKEBACKUP_OPERATION_ID", "op-1")
	destinationProject := getenv("STACKYARD_GCP_GKEBACKUP_DESTINATION_PROJECT", "projects/123456789012")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	clusterName := locationName + "/clusters/" + clusterID
	backupPlanName := locationName + "/backupPlans/" + backupPlanID
	backupChannelName := locationName + "/backupChannels/" + backupChannelID
	backupBindingName := locationName + "/backupPlanBindings/" + backupBindingID
	backupName := backupPlanName + "/backups/" + backupID
	volumeBackupName := backupName + "/volumeBackups/" + volumeBackupID
	restorePlanName := locationName + "/restorePlans/" + restorePlanID
	restoreChannelName := locationName + "/restoreChannels/" + restoreChannelID
	restoreBindingName := locationName + "/restorePlanBindings/" + restoreBindingID
	restoreName := restorePlanName + "/restores/" + restoreID
	volumeRestoreName := restoreName + "/volumeRestores/" + volumeRestoreID
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP GKE Backup apiv1 client using %s\n", apiEndpoint)

	client, err := gkebackup.NewBackupForGKERESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create gkebackup client: %v", err)
	}
	defer closeClient("gkebackup", client.Close)

	calls := []callSpec{
		{
			name: "ListBackupPlans",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListBackupPlans(ctx, &gkebackuppb.ListBackupPlansRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetBackupPlan",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetBackupPlan(ctx, &gkebackuppb.GetBackupPlanRequest{Name: backupPlanName})
				return err
			},
		},
		{
			name: "CreateBackupPlan",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.CreateBackupPlan(ctx, &gkebackuppb.CreateBackupPlanRequest{
					Parent:       locationName,
					BackupPlanId: backupPlanID,
					BackupPlan:   sampleBackupPlan(backupPlanName, clusterName),
				})
				return err
			},
		},
		{
			name: "UpdateBackupPlan",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.UpdateBackupPlan(ctx, &gkebackuppb.UpdateBackupPlanRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
					BackupPlan: &gkebackuppb.BackupPlan{
						Name:        backupPlanName,
						Cluster:     clusterName,
						Description: "updated by stackyard example",
					},
				})
				return err
			},
		},
		{
			name: "DeleteBackupPlan",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.DeleteBackupPlan(ctx, &gkebackuppb.DeleteBackupPlanRequest{Name: backupPlanName})
				return err
			},
		},
		{
			name: "ListBackupChannels",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListBackupChannels(ctx, &gkebackuppb.ListBackupChannelsRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetBackupChannel",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetBackupChannel(ctx, &gkebackuppb.GetBackupChannelRequest{Name: backupChannelName})
				return err
			},
		},
		{
			name: "CreateBackupChannel",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.CreateBackupChannel(ctx, &gkebackuppb.CreateBackupChannelRequest{
					Parent:          locationName,
					BackupChannelId: backupChannelID,
					BackupChannel: &gkebackuppb.BackupChannel{
						Name:               backupChannelName,
						DestinationProject: destinationProject,
						Description:        "stackyard backup channel",
					},
				})
				return err
			},
		},
		{
			name: "UpdateBackupChannel",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.UpdateBackupChannel(ctx, &gkebackuppb.UpdateBackupChannelRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
					BackupChannel: &gkebackuppb.BackupChannel{
						Name:               backupChannelName,
						DestinationProject: destinationProject,
						Description:        "updated by stackyard example",
					},
				})
				return err
			},
		},
		{
			name: "DeleteBackupChannel",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.DeleteBackupChannel(ctx, &gkebackuppb.DeleteBackupChannelRequest{Name: backupChannelName})
				return err
			},
		},
		{
			name: "ListBackupPlanBindings",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListBackupPlanBindings(ctx, &gkebackuppb.ListBackupPlanBindingsRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetBackupPlanBinding",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetBackupPlanBinding(ctx, &gkebackuppb.GetBackupPlanBindingRequest{Name: backupBindingName})
				return err
			},
		},
		{
			name: "ListBackups",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListBackups(ctx, &gkebackuppb.ListBackupsRequest{Parent: backupPlanName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetBackup",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetBackup(ctx, &gkebackuppb.GetBackupRequest{Name: backupName})
				return err
			},
		},
		{
			name: "CreateBackup",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.CreateBackup(ctx, &gkebackuppb.CreateBackupRequest{
					Parent:   backupPlanName,
					BackupId: backupID,
					Backup: &gkebackuppb.Backup{
						Name:        backupName,
						Description: "stackyard backup",
					},
				})
				return err
			},
		},
		{
			name: "UpdateBackup",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.UpdateBackup(ctx, &gkebackuppb.UpdateBackupRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
					Backup: &gkebackuppb.Backup{
						Name:        backupName,
						Description: "updated by stackyard example",
					},
				})
				return err
			},
		},
		{
			name: "DeleteBackup",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.DeleteBackup(ctx, &gkebackuppb.DeleteBackupRequest{Name: backupName})
				return err
			},
		},
		{
			name: "ListVolumeBackups",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListVolumeBackups(ctx, &gkebackuppb.ListVolumeBackupsRequest{Parent: backupName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetVolumeBackup",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetVolumeBackup(ctx, &gkebackuppb.GetVolumeBackupRequest{Name: volumeBackupName})
				return err
			},
		},
		{
			name: "GetBackupIndexDownloadUrl",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetBackupIndexDownloadUrl(ctx, &gkebackuppb.GetBackupIndexDownloadUrlRequest{Backup: backupName})
				return err
			},
		},
		{
			name: "ListRestorePlans",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListRestorePlans(ctx, &gkebackuppb.ListRestorePlansRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetRestorePlan",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetRestorePlan(ctx, &gkebackuppb.GetRestorePlanRequest{Name: restorePlanName})
				return err
			},
		},
		{
			name: "CreateRestorePlan",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.CreateRestorePlan(ctx, &gkebackuppb.CreateRestorePlanRequest{
					Parent:        locationName,
					RestorePlanId: restorePlanID,
					RestorePlan:   sampleRestorePlan(restorePlanName, backupPlanName, clusterName),
				})
				return err
			},
		},
		{
			name: "UpdateRestorePlan",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.UpdateRestorePlan(ctx, &gkebackuppb.UpdateRestorePlanRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
					RestorePlan: &gkebackuppb.RestorePlan{
						Name:          restorePlanName,
						BackupPlan:    backupPlanName,
						Cluster:       clusterName,
						RestoreConfig: sampleRestoreConfig(),
						Description:   "updated by stackyard example",
					},
				})
				return err
			},
		},
		{
			name: "DeleteRestorePlan",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.DeleteRestorePlan(ctx, &gkebackuppb.DeleteRestorePlanRequest{Name: restorePlanName})
				return err
			},
		},
		{
			name: "ListRestoreChannels",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListRestoreChannels(ctx, &gkebackuppb.ListRestoreChannelsRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetRestoreChannel",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetRestoreChannel(ctx, &gkebackuppb.GetRestoreChannelRequest{Name: restoreChannelName})
				return err
			},
		},
		{
			name: "CreateRestoreChannel",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.CreateRestoreChannel(ctx, &gkebackuppb.CreateRestoreChannelRequest{
					Parent:           locationName,
					RestoreChannelId: restoreChannelID,
					RestoreChannel: &gkebackuppb.RestoreChannel{
						Name:               restoreChannelName,
						DestinationProject: destinationProject,
						Description:        "stackyard restore channel",
					},
				})
				return err
			},
		},
		{
			name: "UpdateRestoreChannel",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.UpdateRestoreChannel(ctx, &gkebackuppb.UpdateRestoreChannelRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
					RestoreChannel: &gkebackuppb.RestoreChannel{
						Name:               restoreChannelName,
						DestinationProject: destinationProject,
						Description:        "updated by stackyard example",
					},
				})
				return err
			},
		},
		{
			name: "DeleteRestoreChannel",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.DeleteRestoreChannel(ctx, &gkebackuppb.DeleteRestoreChannelRequest{Name: restoreChannelName})
				return err
			},
		},
		{
			name: "ListRestorePlanBindings",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListRestorePlanBindings(ctx, &gkebackuppb.ListRestorePlanBindingsRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetRestorePlanBinding",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetRestorePlanBinding(ctx, &gkebackuppb.GetRestorePlanBindingRequest{Name: restoreBindingName})
				return err
			},
		},
		{
			name: "ListRestores",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListRestores(ctx, &gkebackuppb.ListRestoresRequest{Parent: restorePlanName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetRestore",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetRestore(ctx, &gkebackuppb.GetRestoreRequest{Name: restoreName})
				return err
			},
		},
		{
			name: "CreateRestore",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.CreateRestore(ctx, &gkebackuppb.CreateRestoreRequest{
					Parent:    restorePlanName,
					RestoreId: restoreID,
					Restore: &gkebackuppb.Restore{
						Name:        restoreName,
						Backup:      backupName,
						Description: "stackyard restore",
					},
				})
				return err
			},
		},
		{
			name: "UpdateRestore",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.UpdateRestore(ctx, &gkebackuppb.UpdateRestoreRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
					Restore: &gkebackuppb.Restore{
						Name:        restoreName,
						Backup:      backupName,
						Description: "updated by stackyard example",
					},
				})
				return err
			},
		},
		{
			name: "DeleteRestore",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.DeleteRestore(ctx, &gkebackuppb.DeleteRestoreRequest{Name: restoreName})
				return err
			},
		},
		{
			name: "ListVolumeRestores",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListVolumeRestores(ctx, &gkebackuppb.ListVolumeRestoresRequest{Parent: restoreName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetVolumeRestore",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetVolumeRestore(ctx, &gkebackuppb.GetVolumeRestoreRequest{Name: volumeRestoreName})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     fmt.Sprintf("projects/%s", projectID),
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
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: backupPlanName})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: backupPlanName,
					Policy:   &iampb.Policy{},
				})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    backupPlanName,
					Permissions: []string{"gkebackup.backupPlans.get"},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *gkebackup.BackupForGKEClient) error {
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

func sampleBackupPlan(name, clusterName string) *gkebackuppb.BackupPlan {
	return &gkebackuppb.BackupPlan{
		Name:        name,
		Cluster:     clusterName,
		Description: "stackyard backup plan",
		RetentionPolicy: &gkebackuppb.BackupPlan_RetentionPolicy{
			BackupDeleteLockDays: 0,
			BackupRetainDays:     7,
		},
		BackupConfig: &gkebackuppb.BackupPlan_BackupConfig{
			BackupScope: &gkebackuppb.BackupPlan_BackupConfig_AllNamespaces{AllNamespaces: true},
		},
	}
}

func sampleRestorePlan(name, backupPlanName, clusterName string) *gkebackuppb.RestorePlan {
	return &gkebackuppb.RestorePlan{
		Name:          name,
		BackupPlan:    backupPlanName,
		Cluster:       clusterName,
		Description:   "stackyard restore plan",
		RestoreConfig: sampleRestoreConfig(),
	}
}

func sampleRestoreConfig() *gkebackuppb.RestoreConfig {
	return &gkebackuppb.RestoreConfig{
		NamespacedResourceRestoreScope: &gkebackuppb.RestoreConfig_AllNamespaces{AllNamespaces: true},
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
