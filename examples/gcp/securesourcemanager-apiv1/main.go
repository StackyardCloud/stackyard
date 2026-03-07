package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	securesourcemanager "cloud.google.com/go/securesourcemanager/apiv1"
	"cloud.google.com/go/securesourcemanager/apiv1/securesourcemanagerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
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
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	instanceID := getenv("STACKYARD_GCP_INSTANCE_ID", "instance-1")
	repositoryID := getenv("STACKYARD_GCP_REPOSITORY_ID", "repository-1")
	hookID := getenv("STACKYARD_GCP_HOOK_ID", "hook-1")
	pullRequestID := getenv("STACKYARD_GCP_PULL_REQUEST_ID", "pull-request-1")
	issueID := getenv("STACKYARD_GCP_ISSUE_ID", "issue-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	instanceName := fmt.Sprintf("%s/instances/%s", locationName, instanceID)
	repositoryName := fmt.Sprintf("%s/repositories/%s", locationName, repositoryID)
	hookName := fmt.Sprintf("%s/hooks/%s", repositoryName, hookID)
	pullRequestName := fmt.Sprintf("%s/pullRequests/%s", repositoryName, pullRequestID)
	issueName := fmt.Sprintf("%s/issues/%s", repositoryName, issueID)

	fmt.Printf("Stackyard GCP Secure Source Manager apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "securesourcemanager",
		},
	}

	client, err := securesourcemanager.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create securesourcemanager client: %v", err)
	}
	defer closeClient(client.Close)

	repositoryOpName := ""

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context) error {
				it := client.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
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
			name: "GetLocation",
			call: func(ctx context.Context) error {
				_, err := client.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListInstances",
			call: func(ctx context.Context) error {
				it := client.ListInstances(ctx, &securesourcemanagerpb.ListInstancesRequest{
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
			name: "GetInstance",
			call: func(ctx context.Context) error {
				_, err := client.GetInstance(ctx, &securesourcemanagerpb.GetInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "ListRepositories",
			call: func(ctx context.Context) error {
				it := client.ListRepositories(ctx, &securesourcemanagerpb.ListRepositoriesRequest{
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
			name: "GetRepository",
			call: func(ctx context.Context) error {
				_, err := client.GetRepository(ctx, &securesourcemanagerpb.GetRepositoryRequest{Name: repositoryName})
				return err
			},
		},
		{
			name: "CreateRepository",
			call: func(ctx context.Context) error {
				op, err := client.CreateRepository(ctx, &securesourcemanagerpb.CreateRepositoryRequest{
					Parent:       locationName,
					RepositoryId: repositoryID,
					Repository: &securesourcemanagerpb.Repository{
						Description: "stackyard repository",
					},
				})
				if err != nil {
					return err
				}
				repositoryOpName = strings.TrimSpace(op.Name())
				return nil
			},
		},
		{
			name: "UpdateRepository",
			call: func(ctx context.Context) error {
				op, err := client.UpdateRepository(ctx, &securesourcemanagerpb.UpdateRepositoryRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
					Repository: &securesourcemanagerpb.Repository{
						Name:        repositoryName,
						Description: "stackyard repository (updated)",
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					repositoryOpName = name
				}
				return nil
			},
		},
		{
			name: "GetRepositoryOperation",
			call: func(ctx context.Context) error {
				if repositoryOpName == "" {
					return nil
				}
				_, err := client.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: repositoryOpName})
				return err
			},
		},
		{
			name: "ListHooks",
			call: func(ctx context.Context) error {
				it := client.ListHooks(ctx, &securesourcemanagerpb.ListHooksRequest{
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
			name: "GetHook",
			call: func(ctx context.Context) error {
				_, err := client.GetHook(ctx, &securesourcemanagerpb.GetHookRequest{Name: hookName})
				return err
			},
		},
		{
			name: "ListPullRequests",
			call: func(ctx context.Context) error {
				it := client.ListPullRequests(ctx, &securesourcemanagerpb.ListPullRequestsRequest{
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
			name: "GetPullRequest",
			call: func(ctx context.Context) error {
				_, err := client.GetPullRequest(ctx, &securesourcemanagerpb.GetPullRequestRequest{Name: pullRequestName})
				return err
			},
		},
		{
			name: "ClosePullRequest",
			call: func(ctx context.Context) error {
				_, err := client.ClosePullRequest(ctx, &securesourcemanagerpb.ClosePullRequestRequest{Name: pullRequestName})
				return err
			},
		},
		{
			name: "ListPullRequestFileDiffs",
			call: func(ctx context.Context) error {
				it := client.ListPullRequestFileDiffs(ctx, &securesourcemanagerpb.ListPullRequestFileDiffsRequest{
					Name:     pullRequestName,
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
			name: "FetchTree",
			call: func(ctx context.Context) error {
				it := client.FetchTree(ctx, &securesourcemanagerpb.FetchTreeRequest{
					Repository: repositoryName,
					Ref:        "main",
					Recursive:  true,
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
			name: "FetchBlob",
			call: func(ctx context.Context) error {
				_, err := client.FetchBlob(ctx, &securesourcemanagerpb.FetchBlobRequest{
					Repository: repositoryName,
					Sha:        "abc123",
				})
				return err
			},
		},
		{
			name: "ListIssues",
			call: func(ctx context.Context) error {
				it := client.ListIssues(ctx, &securesourcemanagerpb.ListIssuesRequest{
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
			name: "GetIssue",
			call: func(ctx context.Context) error {
				_, err := client.GetIssue(ctx, &securesourcemanagerpb.GetIssueRequest{Name: issueName})
				return err
			},
		},
		{
			name: "CloseIssue",
			call: func(ctx context.Context) error {
				_, err := client.CloseIssue(ctx, &securesourcemanagerpb.CloseIssueRequest{Name: issueName})
				return err
			},
		},
		{
			name: "GetIamPolicyRepo",
			call: func(ctx context.Context) error {
				_, err := client.GetIamPolicyRepo(ctx, &iampb.GetIamPolicyRequest{
					Resource: repositoryName,
				})
				return err
			},
		},
		{
			name: "TestIamPermissionsRepo",
			call: func(ctx context.Context) error {
				_, err := client.TestIamPermissionsRepo(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    repositoryName,
					Permissions: []string{"securesourcemanager.repositories.get"},
				})
				return err
			},
		},
		{
			name: "SetIamPolicyRepo",
			call: func(ctx context.Context) error {
				_, err := client.SetIamPolicyRepo(ctx, &iampb.SetIamPolicyRequest{
					Resource: repositoryName,
					Policy: &iampb.Policy{
						Version: 1,
						Bindings: []*iampb.Binding{
							{
								Role:    "roles/securesourcemanager.reader",
								Members: []string{"user:stackyard@example.com"},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := client.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
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

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close securesourcemanager client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
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

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	return base.RoundTrip(clone)
}
