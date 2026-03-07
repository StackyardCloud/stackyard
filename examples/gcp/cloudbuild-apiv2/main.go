package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv2"
	"cloud.google.com/go/cloudbuild/apiv2/cloudbuildpb"
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
	call func(context.Context, *cloudbuild.RepositoryManagerClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	connectionID := getenv("STACKYARD_GCP_CLOUDBUILD_CONNECTION_ID", "team-connection")
	repositoryID := getenv("STACKYARD_GCP_CLOUDBUILD_REPOSITORY_ID", "orders")
	batchRepositoryID := getenv("STACKYARD_GCP_CLOUDBUILD_BATCH_REPOSITORY_ID", "orders-batch")
	operationID := getenv("STACKYARD_GCP_CLOUDBUILD_OPERATION_ID", "op-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	connectionName := locationName + "/connections/" + connectionID
	repositoryName := connectionName + "/repositories/" + repositoryID
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Cloud Build apiv2 client using %s\n", apiEndpoint)

	client, err := cloudbuild.NewRepositoryManagerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloudbuild client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListConnections",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				it := c.ListConnections(ctx, &cloudbuildpb.ListConnectionsRequest{
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
			name: "GetConnection",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.GetConnection(ctx, &cloudbuildpb.GetConnectionRequest{Name: connectionName})
				return err
			},
		},
		{
			name: "CreateConnection",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.CreateConnection(ctx, &cloudbuildpb.CreateConnectionRequest{
					Parent:       locationName,
					Connection:   &cloudbuildpb.Connection{Name: connectionName},
					ConnectionId: connectionID,
				})
				return err
			},
		},
		{
			name: "UpdateConnection",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.UpdateConnection(ctx, &cloudbuildpb.UpdateConnectionRequest{
					Connection: &cloudbuildpb.Connection{Name: connectionName},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
				})
				return err
			},
		},
		{
			name: "DeleteConnection",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.DeleteConnection(ctx, &cloudbuildpb.DeleteConnectionRequest{Name: connectionName})
				return err
			},
		},
		{
			name: "ListRepositories",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				it := c.ListRepositories(ctx, &cloudbuildpb.ListRepositoriesRequest{
					Parent:   connectionName,
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
			name: "GetRepository",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.GetRepository(ctx, &cloudbuildpb.GetRepositoryRequest{Name: repositoryName})
				return err
			},
		},
		{
			name: "CreateRepository",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.CreateRepository(ctx, &cloudbuildpb.CreateRepositoryRequest{
					Parent:       connectionName,
					Repository:   &cloudbuildpb.Repository{Name: repositoryName},
					RepositoryId: repositoryID,
				})
				return err
			},
		},
		{
			name: "BatchCreateRepositories",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.BatchCreateRepositories(ctx, &cloudbuildpb.BatchCreateRepositoriesRequest{
					Parent: connectionName,
					Requests: []*cloudbuildpb.CreateRepositoryRequest{
						{
							Repository:   &cloudbuildpb.Repository{Name: connectionName + "/repositories/" + batchRepositoryID},
							RepositoryId: batchRepositoryID,
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteRepository",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.DeleteRepository(ctx, &cloudbuildpb.DeleteRepositoryRequest{Name: repositoryName})
				return err
			},
		},
		{
			name: "FetchReadToken",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.FetchReadToken(ctx, &cloudbuildpb.FetchReadTokenRequest{Repository: repositoryName})
				return err
			},
		},
		{
			name: "FetchReadWriteToken",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.FetchReadWriteToken(ctx, &cloudbuildpb.FetchReadWriteTokenRequest{Repository: repositoryName})
				return err
			},
		},
		{
			name: "FetchLinkableRepositories",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				it := c.FetchLinkableRepositories(ctx, &cloudbuildpb.FetchLinkableRepositoriesRequest{
					Connection: connectionName,
					PageSize:   1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "FetchGitRefs",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.FetchGitRefs(ctx, &cloudbuildpb.FetchGitRefsRequest{
					Repository: repositoryName,
					RefType:    cloudbuildpb.FetchGitRefsRequest_BRANCH,
				})
				return err
			},
		},
		{
			name: "GetIAMPolicy",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: connectionName})
				return err
			},
		},
		{
			name: "SetIAMPolicy",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: connectionName,
					Policy:   &iampb.Policy{},
				})
				return err
			},
		},
		{
			name: "TestIAMPermissions",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    connectionName,
					Permissions: []string{"cloudbuild.connections.get"},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *cloudbuild.RepositoryManagerClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
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
		fmt.Fprintf(os.Stderr, "warning: close cloudbuild client: %v\n", err)
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
