package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	deploy "cloud.google.com/go/deploy/apiv1"
	"cloud.google.com/go/deploy/apiv1/deploypb"
	iampb "cloud.google.com/go/iam/apiv1/iampb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *deploy.CloudDeployClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	pipelineID := getenv("STACKYARD_GCP_DEPLOY_PIPELINE_ID", "team-pipeline")
	targetID := getenv("STACKYARD_GCP_DEPLOY_TARGET_ID", "team-target")
	releaseID := getenv("STACKYARD_GCP_DEPLOY_RELEASE_ID", "release-1")
	rolloutID := getenv("STACKYARD_GCP_DEPLOY_ROLLOUT_ID", "rollout-1")
	jobRunID := getenv("STACKYARD_GCP_DEPLOY_JOBRUN_ID", "job-1")
	deployPolicyID := getenv("STACKYARD_GCP_DEPLOY_POLICY_ID", "policy-1")
	operationID := getenv("STACKYARD_GCP_DEPLOY_OPERATION_ID", "op-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	pipelineName := locationName + "/deliveryPipelines/" + pipelineID
	targetName := locationName + "/targets/" + targetID
	releaseName := pipelineName + "/releases/" + releaseID
	rolloutName := releaseName + "/rollouts/" + rolloutID
	jobRunName := rolloutName + "/jobRuns/" + jobRunID
	deployPolicyName := locationName + "/deployPolicies/" + deployPolicyID
	operationName := locationName + "/operations/" + operationID
	configName := locationName + "/config"

	fmt.Printf("Stackyard GCP Cloud Deploy apiv1 client using %s\n", apiEndpoint)

	client, err := deploy.NewCloudDeployRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloud deploy client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListDeliveryPipelines",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				it := c.ListDeliveryPipelines(ctx, &deploypb.ListDeliveryPipelinesRequest{
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
			name: "GetDeliveryPipeline",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.GetDeliveryPipeline(ctx, &deploypb.GetDeliveryPipelineRequest{Name: pipelineName})
				return err
			},
		},
		{
			name: "CreateDeliveryPipeline",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.CreateDeliveryPipeline(ctx, &deploypb.CreateDeliveryPipelineRequest{
					Parent:             locationName,
					DeliveryPipelineId: pipelineID,
					DeliveryPipeline:   &deploypb.DeliveryPipeline{Name: pipelineName},
				})
				return err
			},
		},
		{
			name: "UpdateDeliveryPipeline",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.UpdateDeliveryPipeline(ctx, &deploypb.UpdateDeliveryPipelineRequest{
					UpdateMask:       &fieldmaskpb.FieldMask{Paths: []string{"description"}},
					DeliveryPipeline: &deploypb.DeliveryPipeline{Name: pipelineName, Description: "updated by stackyard example"},
				})
				return err
			},
		},
		{
			name: "DeleteDeliveryPipeline",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.DeleteDeliveryPipeline(ctx, &deploypb.DeleteDeliveryPipelineRequest{Name: pipelineName})
				return err
			},
		},
		{
			name: "ListTargets",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				it := c.ListTargets(ctx, &deploypb.ListTargetsRequest{
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
			name: "GetTarget",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.GetTarget(ctx, &deploypb.GetTargetRequest{Name: targetName})
				return err
			},
		},
		{
			name: "CreateTarget",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.CreateTarget(ctx, &deploypb.CreateTargetRequest{
					Parent:   locationName,
					TargetId: targetID,
					Target:   &deploypb.Target{Name: targetName},
				})
				return err
			},
		},
		{
			name: "RollbackTarget",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.RollbackTarget(ctx, &deploypb.RollbackTargetRequest{
					Name: targetName,
				})
				return err
			},
		},
		{
			name: "DeleteTarget",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.DeleteTarget(ctx, &deploypb.DeleteTargetRequest{Name: targetName})
				return err
			},
		},
		{
			name: "ListReleases",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				it := c.ListReleases(ctx, &deploypb.ListReleasesRequest{
					Parent:   pipelineName,
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
			name: "GetRelease",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.GetRelease(ctx, &deploypb.GetReleaseRequest{Name: releaseName})
				return err
			},
		},
		{
			name: "CreateRelease",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.CreateRelease(ctx, &deploypb.CreateReleaseRequest{
					Parent:    pipelineName,
					ReleaseId: releaseID,
					Release:   &deploypb.Release{Name: releaseName},
				})
				return err
			},
		},
		{
			name: "AbandonRelease",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.AbandonRelease(ctx, &deploypb.AbandonReleaseRequest{Name: releaseName})
				return err
			},
		},
		{
			name: "ListRollouts",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				it := c.ListRollouts(ctx, &deploypb.ListRolloutsRequest{
					Parent:   releaseName,
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
			name: "GetRollout",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.GetRollout(ctx, &deploypb.GetRolloutRequest{Name: rolloutName})
				return err
			},
		},
		{
			name: "CreateRollout",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.CreateRollout(ctx, &deploypb.CreateRolloutRequest{
					Parent:    releaseName,
					RolloutId: rolloutID,
					Rollout:   &deploypb.Rollout{Name: rolloutName},
				})
				return err
			},
		},
		{
			name: "ApproveRollout",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.ApproveRollout(ctx, &deploypb.ApproveRolloutRequest{
					Name:     rolloutName,
					Approved: true,
				})
				return err
			},
		},
		{
			name: "AdvanceRollout",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.AdvanceRollout(ctx, &deploypb.AdvanceRolloutRequest{
					Name:    rolloutName,
					PhaseId: "phase-1",
				})
				return err
			},
		},
		{
			name: "CancelRollout",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.CancelRollout(ctx, &deploypb.CancelRolloutRequest{Name: rolloutName})
				return err
			},
		},
		{
			name: "ListJobRuns",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				it := c.ListJobRuns(ctx, &deploypb.ListJobRunsRequest{
					Parent:   rolloutName,
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
			name: "GetJobRun",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.GetJobRun(ctx, &deploypb.GetJobRunRequest{Name: jobRunName})
				return err
			},
		},
		{
			name: "IgnoreJob",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.IgnoreJob(ctx, &deploypb.IgnoreJobRequest{
					Rollout: rolloutName,
					PhaseId: "phase-1",
					JobId:   "deploy",
				})
				return err
			},
		},
		{
			name: "RetryJob",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.RetryJob(ctx, &deploypb.RetryJobRequest{
					Rollout: rolloutName,
					PhaseId: "phase-1",
					JobId:   "deploy",
				})
				return err
			},
		},
		{
			name: "TerminateJobRun",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.TerminateJobRun(ctx, &deploypb.TerminateJobRunRequest{Name: jobRunName})
				return err
			},
		},
		{
			name: "ListDeployPolicies",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				it := c.ListDeployPolicies(ctx, &deploypb.ListDeployPoliciesRequest{
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
			name: "GetDeployPolicy",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.GetDeployPolicy(ctx, &deploypb.GetDeployPolicyRequest{Name: deployPolicyName})
				return err
			},
		},
		{
			name: "GetConfig",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.GetConfig(ctx, &deploypb.GetConfigRequest{Name: configName})
				return err
			},
		},
		{
			name: "GetIAMPolicy",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: pipelineName})
				return err
			},
		},
		{
			name: "SetIAMPolicy",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: pipelineName,
					Policy:   &iampb.Policy{},
				})
				return err
			},
		},
		{
			name: "TestIAMPermissions",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    pipelineName,
					Permissions: []string{"clouddeploy.deliveryPipelines.get"},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
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
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *deploy.CloudDeployClient) error {
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
		fmt.Fprintf(os.Stderr, "warning: close cloud deploy client: %v\n", err)
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
