package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	networkmanagement "cloud.google.com/go/networkmanagement/apiv1"
	"cloud.google.com/go/networkmanagement/apiv1/networkmanagementpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type clients struct {
	reachability *networkmanagement.ReachabilityClient
	vpcFlowLogs  *networkmanagement.VpcFlowLogsClient
}

type callSpec struct {
	name string
	call func(context.Context, *clients) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	testID := getenv("STACKYARD_GCP_NETWORKMANAGEMENT_TEST_ID", "reachability-test-a")
	flowLogsConfigID := getenv("STACKYARD_GCP_NETWORKMANAGEMENT_FLOW_LOGS_CONFIG_ID", "flow-logs-a")

	globalParent := fmt.Sprintf("projects/%s/locations/global", projectID)
	connectivityTestName := fmt.Sprintf("%s/connectivityTests/%s", globalParent, testID)
	vpcFlowLogsConfigName := fmt.Sprintf("%s/vpcFlowLogsConfigs/%s", globalParent, flowLogsConfigID)

	fmt.Printf("Stackyard GCP Network Management apiv1 clients using %s\n", apiEndpoint)

	reachabilityClient, err := networkmanagement.NewReachabilityRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create reachability client: %v", err)
	}
	defer closeClient("reachability", reachabilityClient.Close)

	vpcFlowLogsClient, err := networkmanagement.NewVpcFlowLogsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create vpc flow logs client: %v", err)
	}
	defer closeClient("vpc flow logs", vpcFlowLogsClient.Close)

	allClients := &clients{
		reachability: reachabilityClient,
		vpcFlowLogs:  vpcFlowLogsClient,
	}

	calls := []callSpec{
		{
			name: "ListConnectivityTests",
			call: func(ctx context.Context, c *clients) error {
				it := c.reachability.ListConnectivityTests(ctx, &networkmanagementpb.ListConnectivityTestsRequest{
					Parent:   globalParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetConnectivityTest",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.reachability.GetConnectivityTest(ctx, &networkmanagementpb.GetConnectivityTestRequest{
					Name: connectivityTestName,
				})
				return err
			},
		},
		{
			name: "CreateConnectivityTest",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.reachability.CreateConnectivityTest(ctx, &networkmanagementpb.CreateConnectivityTestRequest{
					Parent: globalParent,
					TestId: testID,
					Resource: &networkmanagementpb.ConnectivityTest{
						Description: "stackyard connectivity test",
						Source:      &networkmanagementpb.Endpoint{IpAddress: "10.1.0.10"},
						Destination: &networkmanagementpb.Endpoint{IpAddress: "10.2.0.20"},
						Protocol:    "TCP",
						Labels:      map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "RerunConnectivityTest",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.reachability.RerunConnectivityTest(ctx, &networkmanagementpb.RerunConnectivityTestRequest{
					Name: connectivityTestName,
				})
				return err
			},
		},
		{
			name: "DeleteConnectivityTest",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.reachability.DeleteConnectivityTest(ctx, &networkmanagementpb.DeleteConnectivityTestRequest{
					Name: connectivityTestName,
				})
				return err
			},
		},
		{
			name: "ListVpcFlowLogsConfigs",
			call: func(ctx context.Context, c *clients) error {
				it := c.vpcFlowLogs.ListVpcFlowLogsConfigs(ctx, &networkmanagementpb.ListVpcFlowLogsConfigsRequest{
					Parent:   globalParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetVpcFlowLogsConfig",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.vpcFlowLogs.GetVpcFlowLogsConfig(ctx, &networkmanagementpb.GetVpcFlowLogsConfigRequest{
					Name: vpcFlowLogsConfigName,
				})
				return err
			},
		},
		{
			name: "CreateVpcFlowLogsConfig",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.vpcFlowLogs.CreateVpcFlowLogsConfig(ctx, &networkmanagementpb.CreateVpcFlowLogsConfigRequest{
					Parent:              globalParent,
					VpcFlowLogsConfigId: flowLogsConfigID,
					VpcFlowLogsConfig: &networkmanagementpb.VpcFlowLogsConfig{
						Description: stringPtr("stackyard flow logs config"),
						TargetResource: &networkmanagementpb.VpcFlowLogsConfig_Network{
							Network: fmt.Sprintf("projects/%s/global/networks/default", projectID),
						},
						Labels: map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteVpcFlowLogsConfig",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.vpcFlowLogs.DeleteVpcFlowLogsConfig(ctx, &networkmanagementpb.DeleteVpcFlowLogsConfigRequest{
					Name: vpcFlowLogsConfigName,
				})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, allClients)
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

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
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
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func stringPtr(s string) *string {
	return &s
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
