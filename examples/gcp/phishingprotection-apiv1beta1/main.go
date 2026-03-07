package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	phishingprotection "cloud.google.com/go/phishingprotection/apiv1beta1"
	"cloud.google.com/go/phishingprotection/apiv1beta1/phishingprotectionpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectNumber := getenv("STACKYARD_GCP_PROJECT_NUMBER", "123456789012")
	parent := fmt.Sprintf("projects/%s", projectNumber)
	uri := getenv("STACKYARD_GCP_PHISHING_URI", "https://example.test/suspected-phishing")

	fmt.Printf("Stackyard GCP Phishing Protection apiv1beta1 client using %s\n", apiEndpoint)

	client, err := phishingprotection.NewPhishingProtectionServiceV1Beta1RESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create phishingprotection client: %v", err)
	}
	defer closeClient("phishingprotection", client.Close)

	_, err = client.ReportPhishing(ctx, &phishingprotectionpb.ReportPhishingRequest{
		Parent: parent,
		Uri:    uri,
	})
	switch {
	case err == nil:
		logf("ReportPhishing succeeded")
	case isToleratedNotImplemented(err):
		logf("ReportPhishing returned NotImplemented (expected in staged emulation)")
	default:
		exitf("ReportPhishing failed: %v", err)
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
