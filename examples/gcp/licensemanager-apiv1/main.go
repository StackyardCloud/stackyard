package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	licensemanager "cloud.google.com/go/licensemanager/apiv1"
	"cloud.google.com/go/licensemanager/apiv1/licensemanagerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type callSpec struct {
	name string
	call func(context.Context, *licensemanager.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	configurationID := getenv("STACKYARD_GCP_LICENSEMANAGER_CONFIGURATION_ID", "config-1")
	instanceID := getenv("STACKYARD_GCP_LICENSEMANAGER_INSTANCE_ID", "instance-1")
	productID := getenv("STACKYARD_GCP_LICENSEMANAGER_PRODUCT_ID", "product-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	projectName := fmt.Sprintf("projects/%s", projectID)
	configurationName := parent + "/configurations/" + configurationID
	instanceName := parent + "/instances/" + instanceID
	productName := parent + "/products/" + productID
	startTime := timestamppb.New(time.Now().UTC().Add(-24 * time.Hour))
	endTime := timestamppb.Now()

	fmt.Printf("Stackyard GCP License Manager apiv1 client using %s\n", apiEndpoint)

	client, err := licensemanager.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create licensemanager client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListConfigurations",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				it := c.ListConfigurations(ctx, &licensemanagerpb.ListConfigurationsRequest{
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
			name: "GetConfiguration",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				_, err := c.GetConfiguration(ctx, &licensemanagerpb.GetConfigurationRequest{Name: configurationName})
				return err
			},
		},
		{
			name: "CreateConfiguration",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				_, err := c.CreateConfiguration(ctx, &licensemanagerpb.CreateConfigurationRequest{
					Parent:          parent,
					ConfigurationId: configurationID,
					Configuration:   baseConfiguration(configurationName, productName),
				})
				return err
			},
		},
		{
			name: "UpdateConfiguration",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				_, err := c.UpdateConfiguration(ctx, &licensemanagerpb.UpdateConfigurationRequest{
					Configuration: &licensemanagerpb.Configuration{
						Name:   configurationName,
						Labels: map[string]string{"owner": "stackyard"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteConfiguration",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				_, err := c.DeleteConfiguration(ctx, &licensemanagerpb.DeleteConfigurationRequest{Name: configurationName})
				return err
			},
		},
		{
			name: "ListInstances",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				it := c.ListInstances(ctx, &licensemanagerpb.ListInstancesRequest{
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
			name: "GetInstance",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				_, err := c.GetInstance(ctx, &licensemanagerpb.GetInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "DeactivateConfiguration",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				_, err := c.DeactivateConfiguration(ctx, &licensemanagerpb.DeactivateConfigurationRequest{Name: configurationName})
				return err
			},
		},
		{
			name: "ReactivateConfiguration",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				_, err := c.ReactivateConfiguration(ctx, &licensemanagerpb.ReactivateConfigurationRequest{Name: configurationName})
				return err
			},
		},
		{
			name: "QueryConfigurationLicenseUsage",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				_, err := c.QueryConfigurationLicenseUsage(ctx, &licensemanagerpb.QueryConfigurationLicenseUsageRequest{
					Name:      configurationName,
					StartTime: startTime,
					EndTime:   endTime,
				})
				return err
			},
		},
		{
			name: "AggregateUsage",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				it := c.AggregateUsage(ctx, &licensemanagerpb.AggregateUsageRequest{
					Name:      configurationName,
					PageSize:  1,
					StartTime: startTime,
					EndTime:   endTime,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "ListProducts",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				it := c.ListProducts(ctx, &licensemanagerpb.ListProductsRequest{
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
			name: "GetProduct",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				_, err := c.GetProduct(ctx, &licensemanagerpb.GetProductRequest{Name: productName})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *licensemanager.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name: projectName,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
	}

	for _, c := range calls {
		err := c.call(ctx, client)
		switch {
		case err == nil:
			logf("%s succeeded", c.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", c.name)
		default:
			exitf("%s failed: %v", c.name, err)
		}
	}

	fmt.Println("Done.")
}

func baseConfiguration(configurationName, productName string) *licensemanagerpb.Configuration {
	return &licensemanagerpb.Configuration{
		Name:        configurationName,
		DisplayName: "Stackyard Configuration",
		Product:     productName,
		LicenseType: licensemanagerpb.LicenseType_LICENSE_TYPE_PER_MONTH_PER_USER,
		CurrentBillingInfo: &licensemanagerpb.BillingInfo{
			CurrentBillingInfo: &licensemanagerpb.BillingInfo_UserCountBilling{
				UserCountBilling: &licensemanagerpb.UserCountBillingInfo{UserCount: 10},
			},
		},
		NextBillingInfo: &licensemanagerpb.BillingInfo{
			CurrentBillingInfo: &licensemanagerpb.BillingInfo_UserCountBilling{
				UserCountBilling: &licensemanagerpb.UserCountBillingInfo{UserCount: 12},
			},
		},
	}
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
		fmt.Fprintf(os.Stderr, "warning: close licensemanager client: %v\n", err)
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
