package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	artifactregistry "cloud.google.com/go/artifactregistry/apiv1"
	"cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *artifactregistry.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	location := getenv("STACKYARD_GCP_LOCATION", "projects/stackyard/locations/us-central1")
	repositoryID := getenv("STACKYARD_GCP_REPOSITORY_ID", "team-repo")
	packageID := getenv("STACKYARD_GCP_PACKAGE_ID", "orders")
	versionID := getenv("STACKYARD_GCP_VERSION_ID", "1.0.0")
	tagID := getenv("STACKYARD_GCP_TAG_ID", "latest")

	repositoryName := location + "/repositories/" + repositoryID
	packageName := repositoryName + "/packages/" + packageID
	versionName := packageName + "/versions/" + versionID
	tagName := packageName + "/tags/" + tagID

	fmt.Printf("Stackyard GCP Artifact Registry apiv1 client using %s\n", apiEndpoint)

	client, err := artifactregistry.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create artifactregistry client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListRepositories",
			call: func(ctx context.Context, c *artifactregistry.Client) error {
				it := c.ListRepositories(ctx, &artifactregistrypb.ListRepositoriesRequest{
					Parent:   location,
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
			call: func(ctx context.Context, c *artifactregistry.Client) error {
				_, err := c.GetRepository(ctx, &artifactregistrypb.GetRepositoryRequest{Name: repositoryName})
				return err
			},
		},
		{
			name: "CreateRepository",
			call: func(ctx context.Context, c *artifactregistry.Client) error {
				_, err := c.CreateRepository(ctx, &artifactregistrypb.CreateRepositoryRequest{
					Parent:       location,
					RepositoryId: repositoryID,
					Repository: &artifactregistrypb.Repository{
						Format:      artifactregistrypb.Repository_DOCKER,
						Description: "Team repository",
					},
				})
				return err
			},
		},
		{
			name: "ListPackages",
			call: func(ctx context.Context, c *artifactregistry.Client) error {
				it := c.ListPackages(ctx, &artifactregistrypb.ListPackagesRequest{
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
			name: "GetPackage",
			call: func(ctx context.Context, c *artifactregistry.Client) error {
				_, err := c.GetPackage(ctx, &artifactregistrypb.GetPackageRequest{Name: packageName})
				return err
			},
		},
		{
			name: "ListVersions",
			call: func(ctx context.Context, c *artifactregistry.Client) error {
				it := c.ListVersions(ctx, &artifactregistrypb.ListVersionsRequest{
					Parent:   packageName,
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
			name: "GetVersion",
			call: func(ctx context.Context, c *artifactregistry.Client) error {
				_, err := c.GetVersion(ctx, &artifactregistrypb.GetVersionRequest{Name: versionName})
				return err
			},
		},
		{
			name: "ListTags",
			call: func(ctx context.Context, c *artifactregistry.Client) error {
				it := c.ListTags(ctx, &artifactregistrypb.ListTagsRequest{
					Parent:   packageName,
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
			name: "GetTag",
			call: func(ctx context.Context, c *artifactregistry.Client) error {
				_, err := c.GetTag(ctx, &artifactregistrypb.GetTagRequest{Name: tagName})
				return err
			},
		},
		{
			name: "ListDockerImages",
			call: func(ctx context.Context, c *artifactregistry.Client) error {
				it := c.ListDockerImages(ctx, &artifactregistrypb.ListDockerImagesRequest{
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
			name: "DeleteRepository",
			call: func(ctx context.Context, c *artifactregistry.Client) error {
				_, err := c.DeleteRepository(ctx, &artifactregistrypb.DeleteRepositoryRequest{Name: repositoryName})
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
		fmt.Fprintf(os.Stderr, "warning: close artifactregistry client: %v\n", err)
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
