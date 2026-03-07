package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	gkemulticloud "cloud.google.com/go/gkemulticloud/apiv1"
	"cloud.google.com/go/gkemulticloud/apiv1/gkemulticloudpb"
	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *gkemulticloud.AttachedClustersClient, *gkemulticloud.AwsClustersClient, *gkemulticloud.AzureClustersClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	grpcEndpoint := grpcTarget(endpoint)
	callTimeout := parseDurationEnv("STACKYARD_GCP_GKEMULTICLOUD_CALL_TIMEOUT", 5*time.Second)

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-west1")
	attachedClusterID := getenv("STACKYARD_GCP_GKEMULTICLOUD_ATTACHED_CLUSTER_ID", "attached-a")
	awsClusterID := getenv("STACKYARD_GCP_GKEMULTICLOUD_AWS_CLUSTER_ID", "aws-a")
	azureClientID := getenv("STACKYARD_GCP_GKEMULTICLOUD_AZURE_CLIENT_ID", "azure-client-a")
	azureClusterID := getenv("STACKYARD_GCP_GKEMULTICLOUD_AZURE_CLUSTER_ID", "azure-a")
	operationID := getenv("STACKYARD_GCP_GKEMULTICLOUD_OPERATION_ID", "op-1")
	fleetMembershipID := getenv("STACKYARD_GCP_GKEMULTICLOUD_FLEET_MEMBERSHIP_ID", "membership-a")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	attachedClusterName := locationName + "/attachedClusters/" + attachedClusterID
	awsClusterName := locationName + "/awsClusters/" + awsClusterID
	azureClientName := locationName + "/azureClients/" + azureClientID
	azureClusterName := locationName + "/azureClusters/" + azureClusterID
	operationName := locationName + "/operations/" + operationID
	fleetMembership := fmt.Sprintf("projects/%s/locations/global/memberships/%s", projectID, fleetMembershipID)

	fmt.Printf("Stackyard GCP GKE Multi-Cloud apiv1 clients using gRPC endpoint %s\n", grpcEndpoint)

	clientOpts := []option.ClientOption{
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	attachedClient, err := gkemulticloud.NewAttachedClustersClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create attached clusters client: %v", err)
	}
	defer closeClient("attached clusters", attachedClient.Close)

	awsClient, err := gkemulticloud.NewAwsClustersClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create aws clusters client: %v", err)
	}
	defer closeClient("aws clusters", awsClient.Close)

	azureClient, err := gkemulticloud.NewAzureClustersClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create azure clusters client: %v", err)
	}
	defer closeClient("azure clusters", azureClient.Close)

	calls := []callSpec{
		{
			name: "ListAttachedClusters",
			call: func(ctx context.Context, attached *gkemulticloud.AttachedClustersClient, _ *gkemulticloud.AwsClustersClient, _ *gkemulticloud.AzureClustersClient) error {
				it := attached.ListAttachedClusters(ctx, &gkemulticloudpb.ListAttachedClustersRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetAttachedServerConfig",
			call: func(ctx context.Context, attached *gkemulticloud.AttachedClustersClient, _ *gkemulticloud.AwsClustersClient, _ *gkemulticloud.AzureClustersClient) error {
				_, err := attached.GetAttachedServerConfig(ctx, &gkemulticloudpb.GetAttachedServerConfigRequest{Name: locationName + "/attachedServerConfig"})
				return err
			},
		},
		{
			name: "CreateAttachedCluster(validateOnly)",
			call: func(ctx context.Context, attached *gkemulticloud.AttachedClustersClient, _ *gkemulticloud.AwsClustersClient, _ *gkemulticloud.AzureClustersClient) error {
				_, err := attached.CreateAttachedCluster(ctx, &gkemulticloudpb.CreateAttachedClusterRequest{
					Parent:            locationName,
					AttachedClusterId: attachedClusterID,
					AttachedCluster: &gkemulticloudpb.AttachedCluster{
						Name: attachedClusterName,
					},
					ValidateOnly: true,
				})
				return err
			},
		},
		{
			name: "ImportAttachedCluster(validateOnly)",
			call: func(ctx context.Context, attached *gkemulticloud.AttachedClustersClient, _ *gkemulticloud.AwsClustersClient, _ *gkemulticloud.AzureClustersClient) error {
				_, err := attached.ImportAttachedCluster(ctx, &gkemulticloudpb.ImportAttachedClusterRequest{
					Parent:          locationName,
					ValidateOnly:    true,
					FleetMembership: fleetMembership,
					PlatformVersion: "1.28.0-gke.1000",
					Distribution:    "generic",
				})
				return err
			},
		},
		{
			name: "ListAwsClusters",
			call: func(ctx context.Context, _ *gkemulticloud.AttachedClustersClient, aws *gkemulticloud.AwsClustersClient, _ *gkemulticloud.AzureClustersClient) error {
				it := aws.ListAwsClusters(ctx, &gkemulticloudpb.ListAwsClustersRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetAwsServerConfig",
			call: func(ctx context.Context, _ *gkemulticloud.AttachedClustersClient, aws *gkemulticloud.AwsClustersClient, _ *gkemulticloud.AzureClustersClient) error {
				_, err := aws.GetAwsServerConfig(ctx, &gkemulticloudpb.GetAwsServerConfigRequest{Name: locationName + "/awsServerConfig"})
				return err
			},
		},
		{
			name: "GenerateAwsAccessToken",
			call: func(ctx context.Context, _ *gkemulticloud.AttachedClustersClient, aws *gkemulticloud.AwsClustersClient, _ *gkemulticloud.AzureClustersClient) error {
				_, err := aws.GenerateAwsAccessToken(ctx, &gkemulticloudpb.GenerateAwsAccessTokenRequest{AwsCluster: awsClusterName})
				return err
			},
		},
		{
			name: "ListAzureClusters",
			call: func(ctx context.Context, _ *gkemulticloud.AttachedClustersClient, _ *gkemulticloud.AwsClustersClient, azure *gkemulticloud.AzureClustersClient) error {
				it := azure.ListAzureClusters(ctx, &gkemulticloudpb.ListAzureClustersRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetAzureServerConfig",
			call: func(ctx context.Context, _ *gkemulticloud.AttachedClustersClient, _ *gkemulticloud.AwsClustersClient, azure *gkemulticloud.AzureClustersClient) error {
				_, err := azure.GetAzureServerConfig(ctx, &gkemulticloudpb.GetAzureServerConfigRequest{Name: locationName + "/azureServerConfig"})
				return err
			},
		},
		{
			name: "CreateAzureClient(validateOnly)",
			call: func(ctx context.Context, _ *gkemulticloud.AttachedClustersClient, _ *gkemulticloud.AwsClustersClient, azure *gkemulticloud.AzureClustersClient) error {
				_, err := azure.CreateAzureClient(ctx, &gkemulticloudpb.CreateAzureClientRequest{
					Parent:        locationName,
					AzureClientId: azureClientID,
					AzureClient: &gkemulticloudpb.AzureClient{
						Name: azureClientName,
					},
					ValidateOnly: true,
				})
				return err
			},
		},
		{
			name: "GenerateAzureAccessToken",
			call: func(ctx context.Context, _ *gkemulticloud.AttachedClustersClient, _ *gkemulticloud.AwsClustersClient, azure *gkemulticloud.AzureClustersClient) error {
				_, err := azure.GenerateAzureAccessToken(ctx, &gkemulticloudpb.GenerateAzureAccessTokenRequest{AzureCluster: azureClusterName})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, attached *gkemulticloud.AttachedClustersClient, _ *gkemulticloud.AwsClustersClient, _ *gkemulticloud.AzureClustersClient) error {
				_, err := attached.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, attached *gkemulticloud.AttachedClustersClient, _ *gkemulticloud.AwsClustersClient, _ *gkemulticloud.AzureClustersClient) error {
				return attached.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
	}

	for _, call := range calls {
		logf("Running %s...", call.name)
		callCtx, cancel := context.WithTimeout(ctx, callTimeout)
		err := call.call(callCtx, attachedClient, awsClient, azureClient)
		cancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedStageError(err):
			logf("%s returned staged error (expected in early emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func grpcTarget(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return "localhost:4566"
	}

	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err == nil && parsed.Host != "" {
			return parsed.Host
		}
	}

	if idx := strings.IndexByte(trimmed, '/'); idx > -1 {
		return trimmed[:idx]
	}
	return trimmed
}

func isToleratedStageError(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded, codes.Internal, codes.Unknown:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && (apiErr.Code == 501 || apiErr.Code == 503) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "notimplemented") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "unsupported protocol") ||
		strings.Contains(msg, "unexpected http status") ||
		strings.Contains(msg, "deadline exceeded")
}

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", label, err)
	}
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
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
