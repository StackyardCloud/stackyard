package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dataplex "cloud.google.com/go/dataplex/apiv1"
	"cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *dataplex.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	lakeID := getenv("STACKYARD_GCP_DATAPLEX_LAKE_ID", "team-lake")
	zoneID := getenv("STACKYARD_GCP_DATAPLEX_ZONE_ID", "raw-zone")
	assetID := getenv("STACKYARD_GCP_DATAPLEX_ASSET_ID", "raw-asset")
	taskID := getenv("STACKYARD_GCP_DATAPLEX_TASK_ID", "profile-task")
	jobID := getenv("STACKYARD_GCP_DATAPLEX_JOB_ID", "job-1")
	environmentID := getenv("STACKYARD_GCP_DATAPLEX_ENVIRONMENT_ID", "analytics")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	lakeName := locationName + "/lakes/" + lakeID
	zoneName := lakeName + "/zones/" + zoneID
	assetName := zoneName + "/assets/" + assetID
	taskName := lakeName + "/tasks/" + taskID
	jobName := taskName + "/jobs/" + jobID
	environmentName := lakeName + "/environments/" + environmentID

	fmt.Printf("Stackyard GCP Dataplex apiv1 client using %s\n", apiEndpoint)

	client, err := dataplex.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create dataplex client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLakes",
			call: func(ctx context.Context, c *dataplex.Client) error {
				it := c.ListLakes(ctx, &dataplexpb.ListLakesRequest{
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
			name: "CreateLake",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.CreateLake(ctx, &dataplexpb.CreateLakeRequest{
					Parent: locationName,
					LakeId: lakeID,
					Lake: &dataplexpb.Lake{
						DisplayName: "Team Lake",
						Description: "Stackyard Dataplex lake",
					},
				})
				return err
			},
		},
		{
			name: "GetLake",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.GetLake(ctx, &dataplexpb.GetLakeRequest{Name: lakeName})
				return err
			},
		},
		{
			name: "UpdateLake",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.UpdateLake(ctx, &dataplexpb.UpdateLakeRequest{
					Lake: &dataplexpb.Lake{
						Name:        lakeName,
						Description: "Stackyard Dataplex lake (updated)",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "ListLakeActions",
			call: func(ctx context.Context, c *dataplex.Client) error {
				it := c.ListLakeActions(ctx, &dataplexpb.ListLakeActionsRequest{
					Parent:   lakeName,
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
			name: "ListZones",
			call: func(ctx context.Context, c *dataplex.Client) error {
				it := c.ListZones(ctx, &dataplexpb.ListZonesRequest{
					Parent:   lakeName,
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
			name: "CreateZone",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.CreateZone(ctx, &dataplexpb.CreateZoneRequest{
					Parent: lakeName,
					ZoneId: zoneID,
					Zone: &dataplexpb.Zone{
						DisplayName: "Raw Zone",
						Type:        dataplexpb.Zone_RAW,
						ResourceSpec: &dataplexpb.Zone_ResourceSpec{
							LocationType: dataplexpb.Zone_ResourceSpec_SINGLE_REGION,
						},
					},
				})
				return err
			},
		},
		{
			name: "GetZone",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.GetZone(ctx, &dataplexpb.GetZoneRequest{Name: zoneName})
				return err
			},
		},
		{
			name: "UpdateZone",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.UpdateZone(ctx, &dataplexpb.UpdateZoneRequest{
					Zone: &dataplexpb.Zone{
						Name:        zoneName,
						Description: "Raw zone for landing data",
						Type:        dataplexpb.Zone_RAW,
						ResourceSpec: &dataplexpb.Zone_ResourceSpec{
							LocationType: dataplexpb.Zone_ResourceSpec_SINGLE_REGION,
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "ListZoneActions",
			call: func(ctx context.Context, c *dataplex.Client) error {
				it := c.ListZoneActions(ctx, &dataplexpb.ListZoneActionsRequest{
					Parent:   zoneName,
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
			name: "ListAssets",
			call: func(ctx context.Context, c *dataplex.Client) error {
				it := c.ListAssets(ctx, &dataplexpb.ListAssetsRequest{
					Parent:   zoneName,
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
			name: "CreateAsset",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.CreateAsset(ctx, &dataplexpb.CreateAssetRequest{
					Parent:  zoneName,
					AssetId: assetID,
					Asset: &dataplexpb.Asset{
						DisplayName: "Raw Asset",
						ResourceSpec: &dataplexpb.Asset_ResourceSpec{
							Name: fmt.Sprintf("projects/%s/datasets/raw_dataset", projectID),
							Type: dataplexpb.Asset_ResourceSpec_BIGQUERY_DATASET,
						},
					},
				})
				return err
			},
		},
		{
			name: "GetAsset",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.GetAsset(ctx, &dataplexpb.GetAssetRequest{Name: assetName})
				return err
			},
		},
		{
			name: "UpdateAsset",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.UpdateAsset(ctx, &dataplexpb.UpdateAssetRequest{
					Asset: &dataplexpb.Asset{
						Name:        assetName,
						Description: "Raw data asset",
						ResourceSpec: &dataplexpb.Asset_ResourceSpec{
							Name: fmt.Sprintf("projects/%s/datasets/raw_dataset", projectID),
							Type: dataplexpb.Asset_ResourceSpec_BIGQUERY_DATASET,
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "ListAssetActions",
			call: func(ctx context.Context, c *dataplex.Client) error {
				it := c.ListAssetActions(ctx, &dataplexpb.ListAssetActionsRequest{
					Parent:   assetName,
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
			name: "ListTasks",
			call: func(ctx context.Context, c *dataplex.Client) error {
				it := c.ListTasks(ctx, &dataplexpb.ListTasksRequest{
					Parent:   lakeName,
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
			name: "CreateTask",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.CreateTask(ctx, &dataplexpb.CreateTaskRequest{
					Parent: lakeName,
					TaskId: taskID,
					Task: &dataplexpb.Task{
						DisplayName: "Profile Task",
						TriggerSpec: &dataplexpb.Task_TriggerSpec{
							Type: dataplexpb.Task_TriggerSpec_ON_DEMAND,
						},
						ExecutionSpec: &dataplexpb.Task_ExecutionSpec{
							ServiceAccount: fmt.Sprintf("stackyard@%s.iam.gserviceaccount.com", projectID),
						},
					},
				})
				return err
			},
		},
		{
			name: "GetTask",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.GetTask(ctx, &dataplexpb.GetTaskRequest{Name: taskName})
				return err
			},
		},
		{
			name: "UpdateTask",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.UpdateTask(ctx, &dataplexpb.UpdateTaskRequest{
					Task: &dataplexpb.Task{
						Name:        taskName,
						Description: "Runs data quality profiling",
						TriggerSpec: &dataplexpb.Task_TriggerSpec{
							Type: dataplexpb.Task_TriggerSpec_ON_DEMAND,
						},
						ExecutionSpec: &dataplexpb.Task_ExecutionSpec{
							ServiceAccount: fmt.Sprintf("stackyard@%s.iam.gserviceaccount.com", projectID),
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "RunTask",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.RunTask(ctx, &dataplexpb.RunTaskRequest{Name: taskName})
				return err
			},
		},
		{
			name: "ListJobs",
			call: func(ctx context.Context, c *dataplex.Client) error {
				it := c.ListJobs(ctx, &dataplexpb.ListJobsRequest{
					Parent:   taskName,
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
			name: "GetJob",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.GetJob(ctx, &dataplexpb.GetJobRequest{Name: jobName})
				return err
			},
		},
		{
			name: "CancelJob",
			call: func(ctx context.Context, c *dataplex.Client) error {
				return c.CancelJob(ctx, &dataplexpb.CancelJobRequest{Name: jobName})
			},
		},
		{
			name: "ListEnvironments",
			call: func(ctx context.Context, c *dataplex.Client) error {
				it := c.ListEnvironments(ctx, &dataplexpb.ListEnvironmentsRequest{
					Parent:   lakeName,
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
			name: "CreateEnvironment",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.CreateEnvironment(ctx, &dataplexpb.CreateEnvironmentRequest{
					Parent:        lakeName,
					EnvironmentId: environmentID,
					Environment: &dataplexpb.Environment{
						DisplayName: "Analytics Environment",
						InfrastructureSpec: &dataplexpb.Environment_InfrastructureSpec{
							Runtime: &dataplexpb.Environment_InfrastructureSpec_OsImage{
								OsImage: &dataplexpb.Environment_InfrastructureSpec_OsImageRuntime{
									ImageVersion: "2.2",
								},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetEnvironment",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.GetEnvironment(ctx, &dataplexpb.GetEnvironmentRequest{Name: environmentName})
				return err
			},
		},
		{
			name: "UpdateEnvironment",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.UpdateEnvironment(ctx, &dataplexpb.UpdateEnvironmentRequest{
					Environment: &dataplexpb.Environment{
						Name:        environmentName,
						Description: "Interactive analytics environment",
						InfrastructureSpec: &dataplexpb.Environment_InfrastructureSpec{
							Runtime: &dataplexpb.Environment_InfrastructureSpec_OsImage{
								OsImage: &dataplexpb.Environment_InfrastructureSpec_OsImageRuntime{
									ImageVersion: "2.2",
								},
							},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "ListSessions",
			call: func(ctx context.Context, c *dataplex.Client) error {
				it := c.ListSessions(ctx, &dataplexpb.ListSessionsRequest{
					Parent:   environmentName,
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
			name: "DeleteAsset",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.DeleteAsset(ctx, &dataplexpb.DeleteAssetRequest{Name: assetName})
				return err
			},
		},
		{
			name: "DeleteTask",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.DeleteTask(ctx, &dataplexpb.DeleteTaskRequest{Name: taskName})
				return err
			},
		},
		{
			name: "DeleteZone",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.DeleteZone(ctx, &dataplexpb.DeleteZoneRequest{Name: zoneName})
				return err
			},
		},
		{
			name: "DeleteEnvironment",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.DeleteEnvironment(ctx, &dataplexpb.DeleteEnvironmentRequest{Name: environmentName})
				return err
			},
		},
		{
			name: "DeleteLake",
			call: func(ctx context.Context, c *dataplex.Client) error {
				_, err := c.DeleteLake(ctx, &dataplexpb.DeleteLakeRequest{Name: lakeName})
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
		fmt.Fprintf(os.Stderr, "warning: close dataplex client: %v\n", err)
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
