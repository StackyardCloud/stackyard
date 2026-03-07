package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dataproc "cloud.google.com/go/dataproc/apiv1"
	"cloud.google.com/go/dataproc/apiv1/dataprocpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
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
	regionID := getenv("STACKYARD_GCP_REGION", "us-central1")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	clusterID := getenv("STACKYARD_GCP_DATAPROC_CLUSTER_ID", "team-cluster")
	jobID := getenv("STACKYARD_GCP_DATAPROC_JOB_ID", "team-job")
	templateID := getenv("STACKYARD_GCP_DATAPROC_TEMPLATE_ID", "team-template")
	policyID := getenv("STACKYARD_GCP_DATAPROC_POLICY_ID", "team-policy")
	batchID := getenv("STACKYARD_GCP_DATAPROC_BATCH_ID", "team-batch")

	regionParent := fmt.Sprintf("projects/%s/regions/%s", projectID, regionID)
	locationParent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	templateName := regionParent + "/workflowTemplates/" + templateID
	policyName := regionParent + "/autoscalingPolicies/" + policyID
	batchName := locationParent + "/batches/" + batchID

	fmt.Printf("Stackyard GCP Dataproc apiv1 clients using %s\n", apiEndpoint)

	clusterClient, err := dataproc.NewClusterControllerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cluster client: %v", err)
	}
	defer closeClient("cluster", clusterClient.Close)

	jobClient, err := dataproc.NewJobControllerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create job client: %v", err)
	}
	defer closeClient("job", jobClient.Close)

	workflowClient, err := dataproc.NewWorkflowTemplateRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create workflow template client: %v", err)
	}
	defer closeClient("workflow template", workflowClient.Close)

	autoscalingClient, err := dataproc.NewAutoscalingPolicyRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create autoscaling policy client: %v", err)
	}
	defer closeClient("autoscaling policy", autoscalingClient.Close)

	batchClient, err := dataproc.NewBatchControllerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create batch client: %v", err)
	}
	defer closeClient("batch", batchClient.Close)

	calls := []callSpec{
		{
			name: "ListClusters",
			call: func(ctx context.Context) error {
				it := clusterClient.ListClusters(ctx, &dataprocpb.ListClustersRequest{
					ProjectId: projectID,
					Region:    regionID,
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
			name: "CreateCluster",
			call: func(ctx context.Context) error {
				_, err := clusterClient.CreateCluster(ctx, &dataprocpb.CreateClusterRequest{
					ProjectId: projectID,
					Region:    regionID,
					Cluster: &dataprocpb.Cluster{
						ClusterName: clusterID,
					},
				})
				return err
			},
		},
		{
			name: "GetCluster",
			call: func(ctx context.Context) error {
				_, err := clusterClient.GetCluster(ctx, &dataprocpb.GetClusterRequest{
					ProjectId:   projectID,
					Region:      regionID,
					ClusterName: clusterID,
				})
				return err
			},
		},
		{
			name: "UpdateCluster",
			call: func(ctx context.Context) error {
				_, err := clusterClient.UpdateCluster(ctx, &dataprocpb.UpdateClusterRequest{
					ProjectId:   projectID,
					Region:      regionID,
					ClusterName: clusterID,
					Cluster: &dataprocpb.Cluster{
						ClusterName: clusterID,
						Labels:      map[string]string{"env": "local"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "StopCluster",
			call: func(ctx context.Context) error {
				_, err := clusterClient.StopCluster(ctx, &dataprocpb.StopClusterRequest{
					ProjectId:   projectID,
					Region:      regionID,
					ClusterName: clusterID,
				})
				return err
			},
		},
		{
			name: "StartCluster",
			call: func(ctx context.Context) error {
				_, err := clusterClient.StartCluster(ctx, &dataprocpb.StartClusterRequest{
					ProjectId:   projectID,
					Region:      regionID,
					ClusterName: clusterID,
				})
				return err
			},
		},
		{
			name: "DiagnoseCluster",
			call: func(ctx context.Context) error {
				_, err := clusterClient.DiagnoseCluster(ctx, &dataprocpb.DiagnoseClusterRequest{
					ProjectId:   projectID,
					Region:      regionID,
					ClusterName: clusterID,
				})
				return err
			},
		},
		{
			name: "DeleteCluster",
			call: func(ctx context.Context) error {
				_, err := clusterClient.DeleteCluster(ctx, &dataprocpb.DeleteClusterRequest{
					ProjectId:   projectID,
					Region:      regionID,
					ClusterName: clusterID,
				})
				return err
			},
		},
		{
			name: "SubmitJob",
			call: func(ctx context.Context) error {
				_, err := jobClient.SubmitJob(ctx, &dataprocpb.SubmitJobRequest{
					ProjectId: projectID,
					Region:    regionID,
					Job: &dataprocpb.Job{
						Reference: &dataprocpb.JobReference{JobId: jobID},
						Placement: &dataprocpb.JobPlacement{ClusterName: clusterID},
						TypeJob: &dataprocpb.Job_HadoopJob{
							HadoopJob: &dataprocpb.HadoopJob{
								Driver: &dataprocpb.HadoopJob_MainClass{MainClass: "example.Main"},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "SubmitJobAsOperation",
			call: func(ctx context.Context) error {
				_, err := jobClient.SubmitJobAsOperation(ctx, &dataprocpb.SubmitJobRequest{
					ProjectId: projectID,
					Region:    regionID,
					Job: &dataprocpb.Job{
						Reference: &dataprocpb.JobReference{JobId: jobID + "-op"},
						Placement: &dataprocpb.JobPlacement{ClusterName: clusterID},
						TypeJob: &dataprocpb.Job_HadoopJob{
							HadoopJob: &dataprocpb.HadoopJob{
								Driver: &dataprocpb.HadoopJob_MainClass{MainClass: "example.Main"},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "ListJobs",
			call: func(ctx context.Context) error {
				it := jobClient.ListJobs(ctx, &dataprocpb.ListJobsRequest{
					ProjectId: projectID,
					Region:    regionID,
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
			name: "GetJob",
			call: func(ctx context.Context) error {
				_, err := jobClient.GetJob(ctx, &dataprocpb.GetJobRequest{
					ProjectId: projectID,
					Region:    regionID,
					JobId:     jobID,
				})
				return err
			},
		},
		{
			name: "UpdateJob",
			call: func(ctx context.Context) error {
				_, err := jobClient.UpdateJob(ctx, &dataprocpb.UpdateJobRequest{
					ProjectId: projectID,
					Region:    regionID,
					JobId:     jobID,
					Job: &dataprocpb.Job{
						Labels: map[string]string{"env": "local"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "CancelJob",
			call: func(ctx context.Context) error {
				_, err := jobClient.CancelJob(ctx, &dataprocpb.CancelJobRequest{
					ProjectId: projectID,
					Region:    regionID,
					JobId:     jobID,
				})
				return err
			},
		},
		{
			name: "DeleteJob",
			call: func(ctx context.Context) error {
				return jobClient.DeleteJob(ctx, &dataprocpb.DeleteJobRequest{
					ProjectId: projectID,
					Region:    regionID,
					JobId:     jobID,
				})
			},
		},
		{
			name: "ListWorkflowTemplates",
			call: func(ctx context.Context) error {
				it := workflowClient.ListWorkflowTemplates(ctx, &dataprocpb.ListWorkflowTemplatesRequest{
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
			name: "CreateWorkflowTemplate",
			call: func(ctx context.Context) error {
				_, err := workflowClient.CreateWorkflowTemplate(ctx, &dataprocpb.CreateWorkflowTemplateRequest{
					Parent: regionParent,
					Template: &dataprocpb.WorkflowTemplate{
						Id: templateID,
					},
				})
				return err
			},
		},
		{
			name: "GetWorkflowTemplate",
			call: func(ctx context.Context) error {
				_, err := workflowClient.GetWorkflowTemplate(ctx, &dataprocpb.GetWorkflowTemplateRequest{
					Name: templateName,
				})
				return err
			},
		},
		{
			name: "UpdateWorkflowTemplate",
			call: func(ctx context.Context) error {
				_, err := workflowClient.UpdateWorkflowTemplate(ctx, &dataprocpb.UpdateWorkflowTemplateRequest{
					Template: &dataprocpb.WorkflowTemplate{
						Name: templateName,
						Id:   templateID,
						Labels: map[string]string{
							"env": "local",
						},
					},
				})
				return err
			},
		},
		{
			name: "InstantiateWorkflowTemplate",
			call: func(ctx context.Context) error {
				_, err := workflowClient.InstantiateWorkflowTemplate(ctx, &dataprocpb.InstantiateWorkflowTemplateRequest{
					Name: templateName,
				})
				return err
			},
		},
		{
			name: "InstantiateInlineWorkflowTemplate",
			call: func(ctx context.Context) error {
				_, err := workflowClient.InstantiateInlineWorkflowTemplate(ctx, &dataprocpb.InstantiateInlineWorkflowTemplateRequest{
					Parent: locationParent,
					Template: &dataprocpb.WorkflowTemplate{
						Id: templateID + "-inline",
					},
				})
				return err
			},
		},
		{
			name: "DeleteWorkflowTemplate",
			call: func(ctx context.Context) error {
				return workflowClient.DeleteWorkflowTemplate(ctx, &dataprocpb.DeleteWorkflowTemplateRequest{
					Name: templateName,
				})
			},
		},
		{
			name: "ListAutoscalingPolicies",
			call: func(ctx context.Context) error {
				it := autoscalingClient.ListAutoscalingPolicies(ctx, &dataprocpb.ListAutoscalingPoliciesRequest{
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
			name: "CreateAutoscalingPolicy",
			call: func(ctx context.Context) error {
				_, err := autoscalingClient.CreateAutoscalingPolicy(ctx, &dataprocpb.CreateAutoscalingPolicyRequest{
					Parent: regionParent,
					Policy: &dataprocpb.AutoscalingPolicy{
						Id: policyID,
						WorkerConfig: &dataprocpb.InstanceGroupAutoscalingPolicyConfig{
							MinInstances: 2,
							MaxInstances: 4,
						},
					},
				})
				return err
			},
		},
		{
			name: "GetAutoscalingPolicy",
			call: func(ctx context.Context) error {
				_, err := autoscalingClient.GetAutoscalingPolicy(ctx, &dataprocpb.GetAutoscalingPolicyRequest{
					Name: policyName,
				})
				return err
			},
		},
		{
			name: "UpdateAutoscalingPolicy",
			call: func(ctx context.Context) error {
				_, err := autoscalingClient.UpdateAutoscalingPolicy(ctx, &dataprocpb.UpdateAutoscalingPolicyRequest{
					Policy: &dataprocpb.AutoscalingPolicy{
						Name: policyName,
						Id:   policyID,
						WorkerConfig: &dataprocpb.InstanceGroupAutoscalingPolicyConfig{
							MinInstances: 2,
							MaxInstances: 6,
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteAutoscalingPolicy",
			call: func(ctx context.Context) error {
				return autoscalingClient.DeleteAutoscalingPolicy(ctx, &dataprocpb.DeleteAutoscalingPolicyRequest{
					Name: policyName,
				})
			},
		},
		{
			name: "ListBatches",
			call: func(ctx context.Context) error {
				it := batchClient.ListBatches(ctx, &dataprocpb.ListBatchesRequest{
					Parent:   locationParent,
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
			name: "CreateBatch",
			call: func(ctx context.Context) error {
				_, err := batchClient.CreateBatch(ctx, &dataprocpb.CreateBatchRequest{
					Parent:  locationParent,
					BatchId: batchID,
					Batch:   &dataprocpb.Batch{},
				})
				return err
			},
		},
		{
			name: "GetBatch",
			call: func(ctx context.Context) error {
				_, err := batchClient.GetBatch(ctx, &dataprocpb.GetBatchRequest{
					Name: batchName,
				})
				return err
			},
		},
		{
			name: "DeleteBatch",
			call: func(ctx context.Context) error {
				return batchClient.DeleteBatch(ctx, &dataprocpb.DeleteBatchRequest{
					Name: batchName,
				})
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
