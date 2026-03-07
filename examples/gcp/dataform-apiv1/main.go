package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dataform "cloud.google.com/go/dataform/apiv1"
	"cloud.google.com/go/dataform/apiv1/dataformpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *dataform.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	repositoryID := getenv("STACKYARD_GCP_DATAFORM_REPOSITORY_ID", "team-repo")
	workspaceID := getenv("STACKYARD_GCP_DATAFORM_WORKSPACE_ID", "dev")
	releaseConfigID := getenv("STACKYARD_GCP_DATAFORM_RELEASE_CONFIG_ID", "daily-release")
	compilationResultID := getenv("STACKYARD_GCP_DATAFORM_COMPILATION_RESULT_ID", "cr-1")
	workflowConfigID := getenv("STACKYARD_GCP_DATAFORM_WORKFLOW_CONFIG_ID", "daily-workflow")
	workflowInvocationID := getenv("STACKYARD_GCP_DATAFORM_WORKFLOW_INVOCATION_ID", "invocation-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	repositoryName := locationName + "/repositories/" + repositoryID
	workspaceName := repositoryName + "/workspaces/" + workspaceID
	releaseConfigName := repositoryName + "/releaseConfigs/" + releaseConfigID
	compilationResultName := repositoryName + "/compilationResults/" + compilationResultID
	workflowConfigName := repositoryName + "/workflowConfigs/" + workflowConfigID
	workflowInvocationName := repositoryName + "/workflowInvocations/" + workflowInvocationID
	configName := locationName + "/config"

	fmt.Printf("Stackyard GCP Dataform apiv1 client using %s\n", apiEndpoint)

	client, err := dataform.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create dataform client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListRepositories",
			call: func(ctx context.Context, c *dataform.Client) error {
				it := c.ListRepositories(ctx, &dataformpb.ListRepositoriesRequest{
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
			name: "CreateRepository",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.CreateRepository(ctx, &dataformpb.CreateRepositoryRequest{
					Parent:       locationName,
					RepositoryId: repositoryID,
					Repository: &dataformpb.Repository{
						DisplayName: "Team Repository",
					},
				})
				return err
			},
		},
		{
			name: "GetRepository",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.GetRepository(ctx, &dataformpb.GetRepositoryRequest{Name: repositoryName})
				return err
			},
		},
		{
			name: "UpdateRepository",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.UpdateRepository(ctx, &dataformpb.UpdateRepositoryRequest{
					Repository: &dataformpb.Repository{
						Name:        repositoryName,
						DisplayName: "Team Repository Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "ListWorkspaces",
			call: func(ctx context.Context, c *dataform.Client) error {
				it := c.ListWorkspaces(ctx, &dataformpb.ListWorkspacesRequest{
					Parent:   repositoryName,
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
			name: "CreateWorkspace",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.CreateWorkspace(ctx, &dataformpb.CreateWorkspaceRequest{
					Parent:      repositoryName,
					WorkspaceId: workspaceID,
					Workspace:   &dataformpb.Workspace{},
				})
				return err
			},
		},
		{
			name: "SearchFiles",
			call: func(ctx context.Context, c *dataform.Client) error {
				it := c.SearchFiles(ctx, &dataformpb.SearchFilesRequest{
					Workspace: workspaceName,
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
			name: "WriteFile",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.WriteFile(ctx, &dataformpb.WriteFileRequest{
					Workspace: workspaceName,
					Path:      "definitions/orders.sqlx",
					Contents:  []byte("select 1 as order_count"),
				})
				return err
			},
		},
		{
			name: "ReadFile",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.ReadFile(ctx, &dataformpb.ReadFileRequest{
					Workspace: workspaceName,
					Path:      "definitions/orders.sqlx",
				})
				return err
			},
		},
		{
			name: "CommitWorkspaceChanges",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.CommitWorkspaceChanges(ctx, &dataformpb.CommitWorkspaceChangesRequest{
					Name: workspaceName,
					Author: &dataformpb.CommitAuthor{
						Name:         "Stackyard",
						EmailAddress: "stackyard@example.com",
					},
					CommitMessage: "initial commit",
				})
				return err
			},
		},
		{
			name: "ListReleaseConfigs",
			call: func(ctx context.Context, c *dataform.Client) error {
				it := c.ListReleaseConfigs(ctx, &dataformpb.ListReleaseConfigsRequest{
					Parent:   repositoryName,
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
			name: "CreateReleaseConfig",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.CreateReleaseConfig(ctx, &dataformpb.CreateReleaseConfigRequest{
					Parent:          repositoryName,
					ReleaseConfigId: releaseConfigID,
					ReleaseConfig: &dataformpb.ReleaseConfig{
						GitCommitish: "main",
					},
				})
				return err
			},
		},
		{
			name: "ListCompilationResults",
			call: func(ctx context.Context, c *dataform.Client) error {
				it := c.ListCompilationResults(ctx, &dataformpb.ListCompilationResultsRequest{
					Parent:   repositoryName,
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
			name: "CreateCompilationResult",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.CreateCompilationResult(ctx, &dataformpb.CreateCompilationResultRequest{
					Parent: repositoryName,
					CompilationResult: &dataformpb.CompilationResult{
						CodeCompilationConfig: &dataformpb.CodeCompilationConfig{
							DefaultDatabase: projectID,
						},
					},
				})
				return err
			},
		},
		{
			name: "QueryCompilationResultActions",
			call: func(ctx context.Context, c *dataform.Client) error {
				it := c.QueryCompilationResultActions(ctx, &dataformpb.QueryCompilationResultActionsRequest{
					Name:     compilationResultName,
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
			name: "ListWorkflowConfigs",
			call: func(ctx context.Context, c *dataform.Client) error {
				it := c.ListWorkflowConfigs(ctx, &dataformpb.ListWorkflowConfigsRequest{
					Parent:   repositoryName,
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
			name: "CreateWorkflowConfig",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.CreateWorkflowConfig(ctx, &dataformpb.CreateWorkflowConfigRequest{
					Parent:           repositoryName,
					WorkflowConfigId: workflowConfigID,
					WorkflowConfig: &dataformpb.WorkflowConfig{
						ReleaseConfig: releaseConfigName,
					},
				})
				return err
			},
		},
		{
			name: "ListWorkflowInvocations",
			call: func(ctx context.Context, c *dataform.Client) error {
				it := c.ListWorkflowInvocations(ctx, &dataformpb.ListWorkflowInvocationsRequest{
					Parent:   repositoryName,
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
			name: "CreateWorkflowInvocation",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.CreateWorkflowInvocation(ctx, &dataformpb.CreateWorkflowInvocationRequest{
					Parent: repositoryName,
					WorkflowInvocation: &dataformpb.WorkflowInvocation{
						CompilationSource: &dataformpb.WorkflowInvocation_WorkflowConfig{
							WorkflowConfig: workflowConfigName,
						},
					},
				})
				return err
			},
		},
		{
			name: "CancelWorkflowInvocation",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.CancelWorkflowInvocation(ctx, &dataformpb.CancelWorkflowInvocationRequest{
					Name: workflowInvocationName,
				})
				return err
			},
		},
		{
			name: "GetConfig",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.GetConfig(ctx, &dataformpb.GetConfigRequest{Name: configName})
				return err
			},
		},
		{
			name: "UpdateConfig",
			call: func(ctx context.Context, c *dataform.Client) error {
				_, err := c.UpdateConfig(ctx, &dataformpb.UpdateConfigRequest{
					Config: &dataformpb.Config{
						Name:              configName,
						DefaultKmsKeyName: "projects/stackyard/locations/us-central1/keyRings/main/cryptoKeys/dataform",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"default_kms_key_name"}},
				})
				return err
			},
		},
		{
			name: "DeleteWorkspace",
			call: func(ctx context.Context, c *dataform.Client) error {
				return c.DeleteWorkspace(ctx, &dataformpb.DeleteWorkspaceRequest{Name: workspaceName})
			},
		},
		{
			name: "DeleteRepository",
			call: func(ctx context.Context, c *dataform.Client) error {
				return c.DeleteRepository(ctx, &dataformpb.DeleteRepositoryRequest{Name: repositoryName})
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
		fmt.Fprintf(os.Stderr, "warning: close dataform client: %v\n", err)
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
