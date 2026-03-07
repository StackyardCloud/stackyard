package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv2"
	"cloud.google.com/go/resourcemanager/apiv2/resourcemanagerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *resourcemanager.FoldersClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	folderParent := getenv("STACKYARD_GCP_FOLDER_PARENT", "organizations/123456")
	folderName := getenv("STACKYARD_GCP_FOLDER_NAME", "folders/1001")
	destinationParent := getenv("STACKYARD_GCP_DESTINATION_PARENT", "folders/2000")

	fmt.Printf("Stackyard GCP Resource Manager apiv2 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "resourcemanager",
		},
	}

	client, err := resourcemanager.NewFoldersRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create resourcemanager client: %v", err)
	}
	defer closeClient(client.Close)

	operationName := "operations/create-folder-1001"

	calls := []callSpec{
		{
			name: "ListFolders",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				it := c.ListFolders(ctx, &resourcemanagerpb.ListFoldersRequest{
					Parent:   folderParent,
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
			name: "SearchFolders",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				it := c.SearchFolders(ctx, &resourcemanagerpb.SearchFoldersRequest{
					Query:    "parent=organizations/123456",
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
			name: "GetFolder",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				_, err := c.GetFolder(ctx, &resourcemanagerpb.GetFolderRequest{Name: folderName})
				return err
			},
		},
		{
			name: "CreateFolder",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				op, err := c.CreateFolder(ctx, &resourcemanagerpb.CreateFolderRequest{
					Parent: folderParent,
					Folder: &resourcemanagerpb.Folder{
						DisplayName: "Team Folder",
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "MoveFolder",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				op, err := c.MoveFolder(ctx, &resourcemanagerpb.MoveFolderRequest{
					Name:              folderName,
					DestinationParent: destinationParent,
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				_, err = op.Poll(ctx)
				return err
			},
		},
		{
			name: "UpdateFolder",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				_, err := c.UpdateFolder(ctx, &resourcemanagerpb.UpdateFolderRequest{
					Folder: &resourcemanagerpb.Folder{
						Name:        folderName,
						DisplayName: "Team Folder Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: folderName})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: folderName,
					Policy: &iampb.Policy{
						Bindings: []*iampb.Binding{{
							Role:    "roles/viewer",
							Members: []string{"user:bob@example.com"},
						}},
					},
				})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    folderName,
					Permissions: []string{"resourcemanager.folders.get", "resourcemanager.folders.update"},
				})
				return err
			},
		},
		{
			name: "PollCreateFolderOperation",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				_, err := c.CreateFolderOperation(operationName).Poll(ctx)
				return err
			},
		},
		{
			name: "DeleteFolder",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				_, err := c.DeleteFolder(ctx, &resourcemanagerpb.DeleteFolderRequest{Name: folderName})
				return err
			},
		},
		{
			name: "UndeleteFolder",
			call: func(ctx context.Context, c *resourcemanager.FoldersClient) error {
				_, err := c.UndeleteFolder(ctx, &resourcemanagerpb.UndeleteFolderRequest{Name: folderName})
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
		fmt.Fprintf(os.Stderr, "warning: close resourcemanager client: %v\n", err)
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
