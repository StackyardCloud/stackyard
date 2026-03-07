package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	cloudprofiler "cloud.google.com/go/cloudprofiler/apiv2"
	"cloud.google.com/go/cloudprofiler/apiv2/cloudprofilerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *cloudprofiler.ProfilerClient, *cloudprofiler.ExportClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	parent := getenv("STACKYARD_GCP_PROFILER_PARENT", "projects/stackyard")
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	profileID := getenv("STACKYARD_GCP_PROFILE_ID", "stackyard-profile")
	profileName := parent + "/profiles/" + profileID

	deployment := &cloudprofilerpb.Deployment{
		ProjectId: projectID,
		Target:    "stackyard-service",
		Labels: map[string]string{
			"language": "go",
			"region":   "us-central1",
		},
	}

	fmt.Printf("Stackyard GCP Cloud Profiler apiv2 clients using %s\n", apiEndpoint)

	profilerClient, err := cloudprofiler.NewProfilerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloudprofiler client: %v", err)
	}
	defer closeClient("cloudprofiler", profilerClient.Close)

	exportClient, err := cloudprofiler.NewExportRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloudprofiler export client: %v", err)
	}
	defer closeClient("cloudprofiler export", exportClient.Close)

	calls := []callSpec{
		{
			name: "ListProfiles",
			call: func(ctx context.Context, _ *cloudprofiler.ProfilerClient, c *cloudprofiler.ExportClient) error {
				it := c.ListProfiles(ctx, &cloudprofilerpb.ListProfilesRequest{
					Parent:   parent,
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
			name: "CreateProfile",
			call: func(ctx context.Context, c *cloudprofiler.ProfilerClient, _ *cloudprofiler.ExportClient) error {
				_, err := c.CreateProfile(ctx, &cloudprofilerpb.CreateProfileRequest{
					Parent:      parent,
					Deployment:  deployment,
					ProfileType: []cloudprofilerpb.ProfileType{cloudprofilerpb.ProfileType_CPU},
				})
				return err
			},
		},
		{
			name: "CreateOfflineProfile",
			call: func(ctx context.Context, c *cloudprofiler.ProfilerClient, _ *cloudprofiler.ExportClient) error {
				_, err := c.CreateOfflineProfile(ctx, &cloudprofilerpb.CreateOfflineProfileRequest{
					Parent: parent,
					Profile: &cloudprofilerpb.Profile{
						Name:         profileName,
						ProfileType:  cloudprofilerpb.ProfileType_CPU,
						Deployment:   deployment,
						ProfileBytes: []byte("stackyard-offline-profile"),
						Labels: map[string]string{
							"source": "stackyard",
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateProfile",
			call: func(ctx context.Context, c *cloudprofiler.ProfilerClient, _ *cloudprofiler.ExportClient) error {
				_, err := c.UpdateProfile(ctx, &cloudprofilerpb.UpdateProfileRequest{
					Profile: &cloudprofilerpb.Profile{
						Name: profileName,
						Labels: map[string]string{
							"updated-by": "stackyard-example",
						},
					},
				})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, profilerClient, exportClient)
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
