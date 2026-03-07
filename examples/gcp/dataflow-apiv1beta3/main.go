package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dataflow "cloud.google.com/go/dataflow/apiv1beta3"
	"cloud.google.com/go/dataflow/apiv1beta3/dataflowpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	jobID := getenv("STACKYARD_GCP_DATAFLOW_JOB_ID", "team-job")
	stageID := getenv("STACKYARD_GCP_DATAFLOW_STAGE_ID", "stage-a")
	snapshotID := getenv("STACKYARD_GCP_DATAFLOW_SNAPSHOT_ID", "snap-1")
	templatePath := getenv("STACKYARD_GCP_DATAFLOW_TEMPLATE_GCS_PATH", "gs://stackyard/templates/wordcount")
	templateJobName := getenv("STACKYARD_GCP_DATAFLOW_TEMPLATE_JOB_NAME", "template-job")
	launchTemplateJobName := getenv("STACKYARD_GCP_DATAFLOW_LAUNCH_TEMPLATE_JOB_NAME", "launch-template-job")
	flexTemplateJobName := getenv("STACKYARD_GCP_DATAFLOW_FLEX_TEMPLATE_JOB_NAME", "flex-template-job")
	flexTemplateSpecPath := getenv("STACKYARD_GCP_DATAFLOW_FLEX_TEMPLATE_SPEC_GCS_PATH", "gs://stackyard/flex/containerSpec.json")

	fmt.Printf("Stackyard GCP Dataflow apiv1beta3 client using %s\n", apiEndpoint)

	jobsClient, err := dataflow.NewJobsV1Beta3RESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create jobs client: %v", err)
	}
	defer closeClient("jobs", jobsClient.Close)

	messagesClient, err := dataflow.NewMessagesV1Beta3RESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create messages client: %v", err)
	}
	defer closeClient("messages", messagesClient.Close)

	metricsClient, err := dataflow.NewMetricsV1Beta3RESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create metrics client: %v", err)
	}
	defer closeClient("metrics", metricsClient.Close)

	snapshotsClient, err := dataflow.NewSnapshotsV1Beta3RESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create snapshots client: %v", err)
	}
	defer closeClient("snapshots", snapshotsClient.Close)

	templatesClient, err := dataflow.NewTemplatesRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create templates client: %v", err)
	}
	defer closeClient("templates", templatesClient.Close)

	flexTemplatesClient, err := dataflow.NewFlexTemplatesRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create flex templates client: %v", err)
	}
	defer closeClient("flex templates", flexTemplatesClient.Close)

	calls := []callSpec{
		{
			name: "CreateJob",
			call: func(ctx context.Context) error {
				_, err := jobsClient.CreateJob(ctx, &dataflowpb.CreateJobRequest{
					ProjectId: projectID,
					Location:  locationID,
					Job: &dataflowpb.Job{
						Name: jobID,
					},
				})
				return err
			},
		},
		{
			name: "GetJob",
			call: func(ctx context.Context) error {
				_, err := jobsClient.GetJob(ctx, &dataflowpb.GetJobRequest{
					ProjectId: projectID,
					Location:  locationID,
					JobId:     jobID,
				})
				return err
			},
		},
		{
			name: "UpdateJob",
			call: func(ctx context.Context) error {
				_, err := jobsClient.UpdateJob(ctx, &dataflowpb.UpdateJobRequest{
					ProjectId: projectID,
					Location:  locationID,
					JobId:     jobID,
					Job: &dataflowpb.Job{
						RequestedState: dataflowpb.JobState_JOB_STATE_CANCELLED,
					},
				})
				return err
			},
		},
		{
			name: "ListJobs",
			call: func(ctx context.Context) error {
				it := jobsClient.ListJobs(ctx, &dataflowpb.ListJobsRequest{
					ProjectId: projectID,
					Location:  locationID,
					PageSize:  1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "AggregatedListJobs",
			call: func(ctx context.Context) error {
				it := jobsClient.AggregatedListJobs(ctx, &dataflowpb.ListJobsRequest{
					ProjectId: projectID,
					PageSize:  1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "SnapshotJob",
			call: func(ctx context.Context) error {
				_, err := jobsClient.SnapshotJob(ctx, &dataflowpb.SnapshotJobRequest{
					ProjectId: projectID,
					Location:  locationID,
					JobId:     jobID,
				})
				return err
			},
		},
		{
			name: "ListJobMessages",
			call: func(ctx context.Context) error {
				it := messagesClient.ListJobMessages(ctx, &dataflowpb.ListJobMessagesRequest{
					ProjectId: projectID,
					Location:  locationID,
					JobId:     jobID,
					PageSize:  1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetJobMetrics",
			call: func(ctx context.Context) error {
				_, err := metricsClient.GetJobMetrics(ctx, &dataflowpb.GetJobMetricsRequest{
					ProjectId: projectID,
					Location:  locationID,
					JobId:     jobID,
				})
				return err
			},
		},
		{
			name: "GetJobExecutionDetails",
			call: func(ctx context.Context) error {
				it := metricsClient.GetJobExecutionDetails(ctx, &dataflowpb.GetJobExecutionDetailsRequest{
					ProjectId: projectID,
					Location:  locationID,
					JobId:     jobID,
					PageSize:  1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetStageExecutionDetails",
			call: func(ctx context.Context) error {
				it := metricsClient.GetStageExecutionDetails(ctx, &dataflowpb.GetStageExecutionDetailsRequest{
					ProjectId: projectID,
					Location:  locationID,
					JobId:     jobID,
					StageId:   stageID,
					PageSize:  1,
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
			call: func(ctx context.Context) error {
				_, err := snapshotsClient.GetSnapshot(ctx, &dataflowpb.GetSnapshotRequest{
					ProjectId:  projectID,
					Location:   locationID,
					SnapshotId: snapshotID,
				})
				return err
			},
		},
		{
			name: "DeleteSnapshot",
			call: func(ctx context.Context) error {
				_, err := snapshotsClient.DeleteSnapshot(ctx, &dataflowpb.DeleteSnapshotRequest{
					ProjectId:  projectID,
					Location:   locationID,
					SnapshotId: snapshotID,
				})
				return err
			},
		},
		{
			name: "ListSnapshots",
			call: func(ctx context.Context) error {
				_, err := snapshotsClient.ListSnapshots(ctx, &dataflowpb.ListSnapshotsRequest{
					ProjectId: projectID,
					Location:  locationID,
					JobId:     jobID,
				})
				return err
			},
		},
		{
			name: "CreateJobFromTemplate",
			call: func(ctx context.Context) error {
				_, err := templatesClient.CreateJobFromTemplate(ctx, &dataflowpb.CreateJobFromTemplateRequest{
					ProjectId: projectID,
					Location:  locationID,
					JobName:   templateJobName,
					Template: &dataflowpb.CreateJobFromTemplateRequest_GcsPath{
						GcsPath: templatePath,
					},
				})
				return err
			},
		},
		{
			name: "LaunchTemplate",
			call: func(ctx context.Context) error {
				_, err := templatesClient.LaunchTemplate(ctx, &dataflowpb.LaunchTemplateRequest{
					ProjectId: projectID,
					Location:  locationID,
					Template: &dataflowpb.LaunchTemplateRequest_GcsPath{
						GcsPath: templatePath,
					},
					LaunchParameters: &dataflowpb.LaunchTemplateParameters{
						JobName: launchTemplateJobName,
					},
				})
				return err
			},
		},
		{
			name: "GetTemplate",
			call: func(ctx context.Context) error {
				_, err := templatesClient.GetTemplate(ctx, &dataflowpb.GetTemplateRequest{
					ProjectId: projectID,
					Location:  locationID,
					Template: &dataflowpb.GetTemplateRequest_GcsPath{
						GcsPath: templatePath,
					},
				})
				return err
			},
		},
		{
			name: "LaunchFlexTemplate",
			call: func(ctx context.Context) error {
				_, err := flexTemplatesClient.LaunchFlexTemplate(ctx, &dataflowpb.LaunchFlexTemplateRequest{
					ProjectId: projectID,
					Location:  locationID,
					LaunchParameter: &dataflowpb.LaunchFlexTemplateParameter{
						JobName: flexTemplateJobName,
						Template: &dataflowpb.LaunchFlexTemplateParameter_ContainerSpecGcsPath{
							ContainerSpecGcsPath: flexTemplateSpecPath,
						},
					},
				})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx)
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
