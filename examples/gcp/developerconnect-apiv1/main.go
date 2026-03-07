package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	developerconnect "cloud.google.com/go/developerconnect/apiv1"
	"cloud.google.com/go/developerconnect/apiv1/developerconnectpb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
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
	connectionID := getenv("STACKYARD_GCP_DEVELOPERCONNECT_CONNECTION_ID", "team-connection")
	repositoryLinkID := getenv("STACKYARD_GCP_DEVELOPERCONNECT_REPOSITORY_LINK_ID", "orders")
	operationID := getenv("STACKYARD_GCP_DEVELOPERCONNECT_OPERATION_ID", "op-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	connectionName := locationName + "/connections/" + connectionID
	repositoryLinkName := connectionName + "/gitRepositoryLinks/" + repositoryLinkID
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Developer Connect apiv1 clients using %s\n", apiEndpoint)

	coreClient, err := developerconnect.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create developerconnect client: %v", err)
	}
	defer closeClient("developerconnect client", coreClient.Close)

	calls := []callSpec{
		{
			name: "ListConnections",
			call: func(ctx context.Context) error {
				it := coreClient.ListConnections(ctx, &developerconnectpb.ListConnectionsRequest{
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
			call: func(ctx context.Context) error {
				_, err := coreClient.GetConnection(ctx, &developerconnectpb.GetConnectionRequest{Name: connectionName})
				return err
			},
		},
		{
			name: "CreateConnection",
			call: func(ctx context.Context) error {
				_, err := coreClient.CreateConnection(ctx, &developerconnectpb.CreateConnectionRequest{
					Parent:       locationName,
					ConnectionId: connectionID,
					Connection:   &developerconnectpb.Connection{Name: connectionName},
				})
				return err
			},
		},
		{
			name: "DeleteConnection",
			call: func(ctx context.Context) error {
				_, err := coreClient.DeleteConnection(ctx, &developerconnectpb.DeleteConnectionRequest{Name: connectionName})
				return err
			},
		},
		{
			name: "ListGitRepositoryLinks",
			call: func(ctx context.Context) error {
				it := coreClient.ListGitRepositoryLinks(ctx, &developerconnectpb.ListGitRepositoryLinksRequest{
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
			name: "GetGitRepositoryLink",
			call: func(ctx context.Context) error {
				_, err := coreClient.GetGitRepositoryLink(ctx, &developerconnectpb.GetGitRepositoryLinkRequest{Name: repositoryLinkName})
				return err
			},
		},
		{
			name: "CreateGitRepositoryLink",
			call: func(ctx context.Context) error {
				_, err := coreClient.CreateGitRepositoryLink(ctx, &developerconnectpb.CreateGitRepositoryLinkRequest{
					Parent:              connectionName,
					GitRepositoryLinkId: repositoryLinkID,
					GitRepositoryLink:   &developerconnectpb.GitRepositoryLink{Name: repositoryLinkName},
				})
				return err
			},
		},
		{
			name: "DeleteGitRepositoryLink",
			call: func(ctx context.Context) error {
				_, err := coreClient.DeleteGitRepositoryLink(ctx, &developerconnectpb.DeleteGitRepositoryLinkRequest{Name: repositoryLinkName})
				return err
			},
		},
		{
			name: "FetchReadToken",
			call: func(ctx context.Context) error {
				_, err := coreClient.FetchReadToken(ctx, &developerconnectpb.FetchReadTokenRequest{
					GitRepositoryLink: repositoryLinkName,
				})
				return err
			},
		},
		{
			name: "FetchReadWriteToken",
			call: func(ctx context.Context) error {
				_, err := coreClient.FetchReadWriteToken(ctx, &developerconnectpb.FetchReadWriteTokenRequest{
					GitRepositoryLink: repositoryLinkName,
				})
				return err
			},
		},
		{
			name: "FetchLinkableGitRepositories",
			call: func(ctx context.Context) error {
				it := coreClient.FetchLinkableGitRepositories(ctx, &developerconnectpb.FetchLinkableGitRepositoriesRequest{
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
			name: "FetchGitHubInstallations",
			call: func(ctx context.Context) error {
				_, err := coreClient.FetchGitHubInstallations(ctx, &developerconnectpb.FetchGitHubInstallationsRequest{
					Connection: connectionName,
				})
				return err
			},
		},
		{
			name: "FetchGitRefs",
			call: func(ctx context.Context) error {
				it := coreClient.FetchGitRefs(ctx, &developerconnectpb.FetchGitRefsRequest{
					GitRepositoryLink: repositoryLinkName,
					RefType:           developerconnectpb.FetchGitRefsRequest_BRANCH,
					PageSize:          1,
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
			call: func(ctx context.Context) error {
				_, err := coreClient.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := coreClient.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
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
			call: func(ctx context.Context) error {
				return coreClient.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context) error {
				return coreClient.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
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

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s: %v\n", label, err)
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
