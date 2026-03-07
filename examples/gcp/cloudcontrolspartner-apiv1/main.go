package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	cloudcontrolspartner "cloud.google.com/go/cloudcontrolspartner/apiv1"
	"cloud.google.com/go/cloudcontrolspartner/apiv1/cloudcontrolspartnerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type coreCallSpec struct {
	name string
	call func(context.Context, *cloudcontrolspartner.CloudControlsPartnerCoreClient) error
}

type monitoringCallSpec struct {
	name string
	call func(context.Context, *cloudcontrolspartner.CloudControlsPartnerMonitoringClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	organization := getenv("STACKYARD_GCP_CLOUDCONTROLSPARTNER_ORGANIZATION", "organizations/123456789")
	location := getenv("STACKYARD_GCP_CLOUDCONTROLSPARTNER_LOCATION", "us-central1")
	customerID := getenv("STACKYARD_GCP_CLOUDCONTROLSPARTNER_CUSTOMER_ID", "team-customer")
	workloadID := getenv("STACKYARD_GCP_CLOUDCONTROLSPARTNER_WORKLOAD_ID", "team-workload")
	violationID := getenv("STACKYARD_GCP_CLOUDCONTROLSPARTNER_VIOLATION_ID", "violation-1")

	locationName := organization + "/locations/" + location
	customerName := locationName + "/customers/" + customerID
	workloadName := customerName + "/workloads/" + workloadID
	ekmConnectionsName := workloadName + "/ekmConnections"
	partnerPermissionsName := workloadName + "/partnerPermissions"
	partnerName := locationName + "/partner"
	violationName := workloadName + "/violations/" + violationID

	fmt.Printf("Stackyard GCP Cloud Controls Partner apiv1 clients using %s\n", apiEndpoint)

	coreClient, err := cloudcontrolspartner.NewCloudControlsPartnerCoreRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloud controls partner core client: %v", err)
	}
	defer closeClient("cloud controls partner core", coreClient.Close)

	monitoringClient, err := cloudcontrolspartner.NewCloudControlsPartnerMonitoringRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloud controls partner monitoring client: %v", err)
	}
	defer closeClient("cloud controls partner monitoring", monitoringClient.Close)

	coreCalls := []coreCallSpec{
		{
			name: "ListCustomers",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerCoreClient) error {
				it := c.ListCustomers(ctx, &cloudcontrolspartnerpb.ListCustomersRequest{
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
			name: "GetCustomer",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerCoreClient) error {
				_, err := c.GetCustomer(ctx, &cloudcontrolspartnerpb.GetCustomerRequest{
					Name: customerName,
				})
				return err
			},
		},
		{
			name: "CreateCustomer",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerCoreClient) error {
				_, err := c.CreateCustomer(ctx, &cloudcontrolspartnerpb.CreateCustomerRequest{
					Parent:     locationName,
					CustomerId: customerID,
					Customer: &cloudcontrolspartnerpb.Customer{
						Name: customerName,
					},
				})
				return err
			},
		},
		{
			name: "UpdateCustomer",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerCoreClient) error {
				_, err := c.UpdateCustomer(ctx, &cloudcontrolspartnerpb.UpdateCustomerRequest{
					Customer: &cloudcontrolspartnerpb.Customer{
						Name: customerName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"display_name"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteCustomer",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerCoreClient) error {
				return c.DeleteCustomer(ctx, &cloudcontrolspartnerpb.DeleteCustomerRequest{
					Name: customerName,
				})
			},
		},
		{
			name: "ListWorkloads",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerCoreClient) error {
				it := c.ListWorkloads(ctx, &cloudcontrolspartnerpb.ListWorkloadsRequest{
					Parent:   customerName,
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
			name: "GetWorkload",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerCoreClient) error {
				_, err := c.GetWorkload(ctx, &cloudcontrolspartnerpb.GetWorkloadRequest{
					Name: workloadName,
				})
				return err
			},
		},
		{
			name: "GetEkmConnections",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerCoreClient) error {
				_, err := c.GetEkmConnections(ctx, &cloudcontrolspartnerpb.GetEkmConnectionsRequest{
					Name: ekmConnectionsName,
				})
				return err
			},
		},
		{
			name: "GetPartnerPermissions",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerCoreClient) error {
				_, err := c.GetPartnerPermissions(ctx, &cloudcontrolspartnerpb.GetPartnerPermissionsRequest{
					Name: partnerPermissionsName,
				})
				return err
			},
		},
		{
			name: "ListAccessApprovalRequests",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerCoreClient) error {
				it := c.ListAccessApprovalRequests(ctx, &cloudcontrolspartnerpb.ListAccessApprovalRequestsRequest{
					Parent:   workloadName,
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
			name: "GetPartner",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerCoreClient) error {
				_, err := c.GetPartner(ctx, &cloudcontrolspartnerpb.GetPartnerRequest{
					Name: partnerName,
				})
				return err
			},
		},
	}

	for _, call := range coreCalls {
		err := call.call(ctx, coreClient)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	monitoringCalls := []monitoringCallSpec{
		{
			name: "ListViolations",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerMonitoringClient) error {
				it := c.ListViolations(ctx, &cloudcontrolspartnerpb.ListViolationsRequest{
					Parent:   workloadName,
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
			name: "GetViolation",
			call: func(ctx context.Context, c *cloudcontrolspartner.CloudControlsPartnerMonitoringClient) error {
				_, err := c.GetViolation(ctx, &cloudcontrolspartnerpb.GetViolationRequest{
					Name: violationName,
				})
				return err
			},
		},
	}

	for _, call := range monitoringCalls {
		err := call.call(ctx, monitoringClient)
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
