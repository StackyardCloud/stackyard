package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	channel "cloud.google.com/go/channel/apiv1"
	"cloud.google.com/go/channel/apiv1/channelpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type channelCallSpec struct {
	name string
	call func(context.Context, *channel.CloudChannelClient) error
}

type reportsCallSpec struct {
	name string
	call func(context.Context, *channel.CloudChannelReportsClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	account := getenv("STACKYARD_GCP_CHANNEL_ACCOUNT", "accounts/stackyard-account")
	customerID := getenv("STACKYARD_GCP_CHANNEL_CUSTOMER_ID", "team-customer")
	reportID := getenv("STACKYARD_GCP_CHANNEL_REPORT_ID", "613bf59q")
	reportJobID := getenv("STACKYARD_GCP_CHANNEL_REPORT_JOB_ID", "team-report-job")

	customerName := account + "/customers/" + customerID
	reportName := account + "/reports/" + reportID
	reportJobName := account + "/reportJobs/" + reportJobID

	fmt.Printf("Stackyard GCP Cloud Channel apiv1 clients using %s\n", apiEndpoint)

	channelClient, err := channel.NewCloudChannelRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloud channel client: %v", err)
	}
	defer closeClient("cloud channel", channelClient.Close)

	reportsClient, err := channel.NewCloudChannelReportsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloud channel reports client: %v", err)
	}
	defer closeClient("cloud channel reports", reportsClient.Close)

	channelCalls := []channelCallSpec{
		{
			name: "ListCustomers",
			call: func(ctx context.Context, c *channel.CloudChannelClient) error {
				it := c.ListCustomers(ctx, &channelpb.ListCustomersRequest{
					Parent:   account,
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
			call: func(ctx context.Context, c *channel.CloudChannelClient) error {
				_, err := c.GetCustomer(ctx, &channelpb.GetCustomerRequest{
					Name: customerName,
				})
				return err
			},
		},
		{
			name: "CheckCloudIdentityAccountsExist",
			call: func(ctx context.Context, c *channel.CloudChannelClient) error {
				_, err := c.CheckCloudIdentityAccountsExist(ctx, &channelpb.CheckCloudIdentityAccountsExistRequest{
					Parent: account,
					Domain: "example.com",
				})
				return err
			},
		},
		{
			name: "ListProducts",
			call: func(ctx context.Context, c *channel.CloudChannelClient) error {
				it := c.ListProducts(ctx, &channelpb.ListProductsRequest{
					Account:  account,
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
			name: "ListOffers",
			call: func(ctx context.Context, c *channel.CloudChannelClient) error {
				it := c.ListOffers(ctx, &channelpb.ListOffersRequest{
					Parent:   account,
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
			name: "ListEntitlements",
			call: func(ctx context.Context, c *channel.CloudChannelClient) error {
				it := c.ListEntitlements(ctx, &channelpb.ListEntitlementsRequest{
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
	}

	for _, call := range channelCalls {
		err := call.call(ctx, channelClient)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	reportsCalls := []reportsCallSpec{
		{
			name: "ListReports",
			call: func(ctx context.Context, c *channel.CloudChannelReportsClient) error {
				it := c.ListReports(ctx, &channelpb.ListReportsRequest{
					Parent:   account,
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
			name: "RunReportJob",
			call: func(ctx context.Context, c *channel.CloudChannelReportsClient) error {
				_, err := c.RunReportJob(ctx, &channelpb.RunReportJobRequest{
					Name: reportName,
				})
				return err
			},
		},
		{
			name: "FetchReportResults",
			call: func(ctx context.Context, c *channel.CloudChannelReportsClient) error {
				it := c.FetchReportResults(ctx, &channelpb.FetchReportResultsRequest{
					ReportJob: reportJobName,
					PageSize:  1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
	}

	for _, call := range reportsCalls {
		err := call.call(ctx, reportsClient)
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
