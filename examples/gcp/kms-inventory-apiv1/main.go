package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	inventory "cloud.google.com/go/kms/inventory/apiv1"
	"cloud.google.com/go/kms/inventory/apiv1/inventorypb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	organizationID := getenv("STACKYARD_GCP_ORGANIZATION_ID", "123456789")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	keyRingID := getenv("STACKYARD_GCP_KMS_KEY_RING_ID", "team-ring")
	cryptoKeyID := getenv("STACKYARD_GCP_KMS_CRYPTO_KEY_ID", "app-key")

	projectName := fmt.Sprintf("projects/%s", projectID)
	organizationName := fmt.Sprintf("organizations/%s", organizationID)
	cryptoKeyName := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", projectID, locationID, keyRingID, cryptoKeyID)

	fmt.Printf("Stackyard GCP Cloud KMS Inventory apiv1 clients using %s\n", apiEndpoint)

	keyDashboardClient, err := inventory.NewKeyDashboardRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create key dashboard client: %v", err)
	}
	defer closeClient("key dashboard", keyDashboardClient.Close)

	keyTrackingClient, err := inventory.NewKeyTrackingRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create key tracking client: %v", err)
	}
	defer closeClient("key tracking", keyTrackingClient.Close)

	calls := []callSpec{
		{
			name: "ListCryptoKeys",
			call: func(ctx context.Context) error {
				it := keyDashboardClient.ListCryptoKeys(ctx, &inventorypb.ListCryptoKeysRequest{
					Parent:   projectName,
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
			name: "GetProtectedResourcesSummary",
			call: func(ctx context.Context) error {
				_, err := keyTrackingClient.GetProtectedResourcesSummary(ctx, &inventorypb.GetProtectedResourcesSummaryRequest{
					Name: cryptoKeyName,
				})
				return err
			},
		},
		{
			name: "SearchProtectedResources",
			call: func(ctx context.Context) error {
				it := keyTrackingClient.SearchProtectedResources(ctx, &inventorypb.SearchProtectedResourcesRequest{
					Scope:         organizationName,
					CryptoKey:     cryptoKeyName,
					PageSize:      1,
					ResourceTypes: []string{"compute.googleapis.com/Disk"},
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx)
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
