package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	policytroubleshooter "cloud.google.com/go/policytroubleshooter/apiv1"
	"cloud.google.com/go/policytroubleshooter/apiv1/policytroubleshooterpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	principal := getenv("STACKYARD_GCP_PRINCIPAL", "alice@example.com")
	fullResourceName := getenv("STACKYARD_GCP_FULL_RESOURCE_NAME", "//cloudresourcemanager.googleapis.com/projects/stackyard")
	permission := getenv("STACKYARD_GCP_PERMISSION", "resourcemanager.projects.get")

	fmt.Printf("Stackyard GCP Policy Troubleshooter apiv1 client using %s\n", apiEndpoint)

	client, err := policytroubleshooter.NewIamCheckerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create policytroubleshooter client: %v", err)
	}
	defer closeClient("policytroubleshooter", client.Close)

	_, err = client.TroubleshootIamPolicy(ctx, &policytroubleshooterpb.TroubleshootIamPolicyRequest{
		AccessTuple: &policytroubleshooterpb.AccessTuple{
			Principal:        principal,
			FullResourceName: fullResourceName,
			Permission:       permission,
		},
	})
	switch {
	case err == nil:
		logf("TroubleshootIamPolicy succeeded")
	case isToleratedNotImplemented(err):
		logf("TroubleshootIamPolicy returned NotImplemented (expected in staged emulation)")
	default:
		exitf("TroubleshootIamPolicy failed: %v", err)
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

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "notimplemented") || strings.Contains(lower, "not implemented")
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
