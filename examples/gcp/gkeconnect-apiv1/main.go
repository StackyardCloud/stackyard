package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	gateway "cloud.google.com/go/gkeconnect/gateway/apiv1"
	"cloud.google.com/go/gkeconnect/gateway/apiv1/gatewaypb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	req  *gatewaypb.GenerateCredentialsRequest
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	membershipID := getenv("STACKYARD_GCP_GKECONNECT_MEMBERSHIP_ID", "cluster-a")

	membershipName := fmt.Sprintf("projects/%s/locations/%s/memberships/%s", projectID, locationID, membershipID)

	fmt.Printf("Stackyard GCP Connect Gateway apiv1 client using %s\n", apiEndpoint)

	client, err := gateway.NewGatewayControlRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create gkeconnect gateway client: %v", err)
	}
	defer closeClient("gkeconnect gateway client", client.Close)

	calls := []callSpec{
		{
			name: "GenerateCredentials(default)",
			req: &gatewaypb.GenerateCredentialsRequest{
				Name: membershipName,
			},
		},
		{
			name: "GenerateCredentials(with-options)",
			req: &gatewaypb.GenerateCredentialsRequest{
				Name:                membershipName,
				ForceUseAgent:       true,
				Version:             "v1",
				KubernetesNamespace: "default",
				OperatingSystem:     gatewaypb.GenerateCredentialsRequest_OPERATING_SYSTEM_WINDOWS,
			},
		},
	}

	for _, call := range calls {
		resp, err := client.GenerateCredentials(ctx, call.req)
		switch {
		case err == nil:
			logf("%s succeeded endpoint=%q kubeconfig_bytes=%d", call.name, resp.GetEndpoint(), len(resp.GetKubeconfig()))
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

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s: %v\n", label, err)
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
