package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	configdelivery "cloud.google.com/go/configdelivery/apiv1"
	"cloud.google.com/go/configdelivery/apiv1/configdeliverypb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *configdelivery.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_CONFIGDELIVERY_LOCATION", "us-central1")
	resourceBundleID := getenv("STACKYARD_GCP_CONFIGDELIVERY_RESOURCE_BUNDLE_ID", "platform-bundle")
	releaseID := getenv("STACKYARD_GCP_CONFIGDELIVERY_RELEASE_ID", "r-1")
	variantID := getenv("STACKYARD_GCP_CONFIGDELIVERY_VARIANT_ID", "default")
	fleetPackageID := getenv("STACKYARD_GCP_CONFIGDELIVERY_FLEET_PACKAGE_ID", "platform-package")
	rolloutID := getenv("STACKYARD_GCP_CONFIGDELIVERY_ROLLOUT_ID", "rollout-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	resourceBundleName := fmt.Sprintf("%s/resourceBundles/%s", locationName, resourceBundleID)
	releaseName := fmt.Sprintf("%s/releases/%s", resourceBundleName, releaseID)
	variantName := fmt.Sprintf("%s/variants/%s", releaseName, variantID)
	fleetPackageName := fmt.Sprintf("%s/fleetPackages/%s", locationName, fleetPackageID)
	rolloutName := fmt.Sprintf("%s/rollouts/%s", fleetPackageName, rolloutID)

	fmt.Printf("Stackyard GCP Config Delivery apiv1 client using %s\n", apiEndpoint)

	client, err := configdelivery.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create configdelivery client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *configdelivery.Client) error {
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
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListResourceBundles",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				it := c.ListResourceBundles(ctx, &configdeliverypb.ListResourceBundlesRequest{
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
			name: "GetResourceBundle",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.GetResourceBundle(ctx, &configdeliverypb.GetResourceBundleRequest{Name: resourceBundleName})
				return err
			},
		},
		{
			name: "CreateResourceBundle",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.CreateResourceBundle(ctx, &configdeliverypb.CreateResourceBundleRequest{
					Parent:           locationName,
					ResourceBundleId: resourceBundleID,
					ResourceBundle:   &configdeliverypb.ResourceBundle{},
				})
				return err
			},
		},
		{
			name: "DeleteResourceBundle",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.DeleteResourceBundle(ctx, &configdeliverypb.DeleteResourceBundleRequest{Name: resourceBundleName})
				return err
			},
		},
		{
			name: "ListFleetPackages",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				it := c.ListFleetPackages(ctx, &configdeliverypb.ListFleetPackagesRequest{
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
			name: "GetFleetPackage",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.GetFleetPackage(ctx, &configdeliverypb.GetFleetPackageRequest{Name: fleetPackageName})
				return err
			},
		},
		{
			name: "CreateFleetPackage",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.CreateFleetPackage(ctx, &configdeliverypb.CreateFleetPackageRequest{
					Parent:         locationName,
					FleetPackageId: fleetPackageID,
					FleetPackage: &configdeliverypb.FleetPackage{
						ResourceBundleSelector: &configdeliverypb.FleetPackage_ResourceBundleSelector{
							Source: &configdeliverypb.FleetPackage_ResourceBundleSelector_ResourceBundle{
								ResourceBundle: &configdeliverypb.FleetPackage_ResourceBundleTag{
									Name: resourceBundleName,
									Tag:  "v1.0.0",
								},
							},
						},
						VariantSelector: &configdeliverypb.FleetPackage_VariantSelector{
							Strategy: &configdeliverypb.FleetPackage_VariantSelector_VariantNameTemplate{
								VariantNameTemplate: "default",
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteFleetPackage",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.DeleteFleetPackage(ctx, &configdeliverypb.DeleteFleetPackageRequest{Name: fleetPackageName})
				return err
			},
		},
		{
			name: "ListReleases",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				it := c.ListReleases(ctx, &configdeliverypb.ListReleasesRequest{
					Parent:   resourceBundleName,
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
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.GetRelease(ctx, &configdeliverypb.GetReleaseRequest{Name: releaseName})
				return err
			},
		},
		{
			name: "CreateRelease",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.CreateRelease(ctx, &configdeliverypb.CreateReleaseRequest{
					Parent:    resourceBundleName,
					ReleaseId: releaseID,
					Release: &configdeliverypb.Release{
						Version: "v1.0.0",
					},
				})
				return err
			},
		},
		{
			name: "DeleteRelease",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.DeleteRelease(ctx, &configdeliverypb.DeleteReleaseRequest{Name: releaseName})
				return err
			},
		},
		{
			name: "ListVariants",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				it := c.ListVariants(ctx, &configdeliverypb.ListVariantsRequest{
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
			name: "GetVariant",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.GetVariant(ctx, &configdeliverypb.GetVariantRequest{Name: variantName})
				return err
			},
		},
		{
			name: "CreateVariant",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.CreateVariant(ctx, &configdeliverypb.CreateVariantRequest{
					Parent:    releaseName,
					VariantId: variantID,
					Variant: &configdeliverypb.Variant{
						Resources: []string{
							"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: sample\n",
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteVariant",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.DeleteVariant(ctx, &configdeliverypb.DeleteVariantRequest{Name: variantName})
				return err
			},
		},
		{
			name: "ListRollouts",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				it := c.ListRollouts(ctx, &configdeliverypb.ListRolloutsRequest{
					Parent:   fleetPackageName,
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
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.GetRollout(ctx, &configdeliverypb.GetRolloutRequest{Name: rolloutName})
				return err
			},
		},
		{
			name: "SuspendRollout",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.SuspendRollout(ctx, &configdeliverypb.SuspendRolloutRequest{
					Name:   rolloutName,
					Reason: "pause rollout for verification",
				})
				return err
			},
		},
		{
			name: "ResumeRollout",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.ResumeRollout(ctx, &configdeliverypb.ResumeRolloutRequest{
					Name:   rolloutName,
					Reason: "resume rollout after verification",
				})
				return err
			},
		},
		{
			name: "AbortRollout",
			call: func(ctx context.Context, c *configdelivery.Client) error {
				_, err := c.AbortRollout(ctx, &configdeliverypb.AbortRolloutRequest{
					Name:   rolloutName,
					Reason: "abort rollout for staged emulation test",
				})
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
		fmt.Fprintf(os.Stderr, "warning: close configdelivery client: %v\n", err)
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
