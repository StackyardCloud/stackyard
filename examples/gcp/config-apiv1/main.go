package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	config "cloud.google.com/go/config/apiv1"
	"cloud.google.com/go/config/apiv1/configpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *config.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_CONFIG_LOCATION", "us-central1")
	deploymentID := getenv("STACKYARD_GCP_CONFIG_DEPLOYMENT_ID", "platform-foundation")
	previewID := getenv("STACKYARD_GCP_CONFIG_PREVIEW_ID", "preview-1")
	terraformVersionID := getenv("STACKYARD_GCP_CONFIG_TERRAFORM_VERSION_ID", "1-6-6")
	resourceChangeID := getenv("STACKYARD_GCP_CONFIG_RESOURCE_CHANGE_ID", "resource-change-1")
	resourceDriftID := getenv("STACKYARD_GCP_CONFIG_RESOURCE_DRIFT_ID", "resource-drift-1")
	serviceAccount := getenv("STACKYARD_GCP_CONFIG_SERVICE_ACCOUNT", "projects/stackyard/serviceAccounts/stackyard@stackyard.iam.gserviceaccount.com")

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	deploymentName := fmt.Sprintf("%s/deployments/%s", locationName, deploymentID)
	previewName := fmt.Sprintf("%s/previews/%s", locationName, previewID)
	terraformVersionName := fmt.Sprintf("%s/terraformVersions/%s", locationName, terraformVersionID)
	resourceChangeName := fmt.Sprintf("%s/resourceChanges/%s", previewName, resourceChangeID)
	resourceDriftName := fmt.Sprintf("%s/resourceDrifts/%s", previewName, resourceDriftID)

	fmt.Printf("Stackyard GCP Infrastructure Manager (config apiv1) client using %s\n", apiEndpoint)

	client, err := config.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
		option.WithUserAgent("stackyard-config-apiv1"),
	)
	if err != nil {
		exitf("failed to create config client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *config.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
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
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListDeployments",
			call: func(ctx context.Context, c *config.Client) error {
				it := c.ListDeployments(ctx, &configpb.ListDeploymentsRequest{
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
			name: "GetDeployment",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.GetDeployment(ctx, &configpb.GetDeploymentRequest{Name: deploymentName})
				return err
			},
		},
		{
			name: "CreateDeployment",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.CreateDeployment(ctx, &configpb.CreateDeploymentRequest{
					Parent:       locationName,
					DeploymentId: deploymentID,
					Deployment: &configpb.Deployment{
						ServiceAccount: stringPtr(serviceAccount),
					},
				})
				return err
			},
		},
		{
			name: "DeleteDeployment",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.DeleteDeployment(ctx, &configpb.DeleteDeploymentRequest{Name: deploymentName})
				return err
			},
		},
		{
			name: "LockDeployment",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.LockDeployment(ctx, &configpb.LockDeploymentRequest{Name: deploymentName})
				return err
			},
		},
		{
			name: "UnlockDeployment",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.UnlockDeployment(ctx, &configpb.UnlockDeploymentRequest{
					Name:   deploymentName,
					LockId: 1,
				})
				return err
			},
		},
		{
			name: "ExportLockInfo",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.ExportLockInfo(ctx, &configpb.ExportLockInfoRequest{Name: deploymentName})
				return err
			},
		},
		{
			name: "ListPreviews",
			call: func(ctx context.Context, c *config.Client) error {
				it := c.ListPreviews(ctx, &configpb.ListPreviewsRequest{
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
			name: "GetPreview",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.GetPreview(ctx, &configpb.GetPreviewRequest{Name: previewName})
				return err
			},
		},
		{
			name: "CreatePreview",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.CreatePreview(ctx, &configpb.CreatePreviewRequest{
					Parent:    locationName,
					PreviewId: previewID,
					Preview: &configpb.Preview{
						ServiceAccount: serviceAccount,
					},
				})
				return err
			},
		},
		{
			name: "DeletePreview",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.DeletePreview(ctx, &configpb.DeletePreviewRequest{Name: previewName})
				return err
			},
		},
		{
			name: "ExportPreviewResult",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.ExportPreviewResult(ctx, &configpb.ExportPreviewResultRequest{Parent: previewName})
				return err
			},
		},
		{
			name: "ListTerraformVersions",
			call: func(ctx context.Context, c *config.Client) error {
				it := c.ListTerraformVersions(ctx, &configpb.ListTerraformVersionsRequest{
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
			name: "GetTerraformVersion",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.GetTerraformVersion(ctx, &configpb.GetTerraformVersionRequest{Name: terraformVersionName})
				return err
			},
		},
		{
			name: "ListResourceChanges",
			call: func(ctx context.Context, c *config.Client) error {
				it := c.ListResourceChanges(ctx, &configpb.ListResourceChangesRequest{
					Parent:   previewName,
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
			name: "GetResourceChange",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.GetResourceChange(ctx, &configpb.GetResourceChangeRequest{Name: resourceChangeName})
				return err
			},
		},
		{
			name: "ListResourceDrifts",
			call: func(ctx context.Context, c *config.Client) error {
				it := c.ListResourceDrifts(ctx, &configpb.ListResourceDriftsRequest{
					Parent:   previewName,
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
			name: "GetResourceDrift",
			call: func(ctx context.Context, c *config.Client) error {
				_, err := c.GetResourceDrift(ctx, &configpb.GetResourceDriftRequest{Name: resourceDriftName})
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
		fmt.Fprintf(os.Stderr, "warning: close config client: %v\n", err)
	}
}

func stringPtr(v string) *string {
	return &v
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
