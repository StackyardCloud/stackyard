package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	publicca "cloud.google.com/go/security/publicca/apiv1"
	"cloud.google.com/go/security/publicca/apiv1/publiccapb"
	"google.golang.org/api/googleapi"
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
	locationID := getenv("STACKYARD_GCP_LOCATION", "global")
	externalAccountKeyID := getenv("STACKYARD_GCP_EXTERNAL_ACCOUNT_KEY_ID", "eak-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	externalAccountKeyName := fmt.Sprintf("%s/externalAccountKeys/%s", parent, externalAccountKeyID)

	fmt.Printf("Stackyard GCP Public Certificate Authority security/publicca/apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "security-publicca",
		},
	}

	client, err := publicca.NewPublicCertificateAuthorityRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create security publicca client: %v", err)
	}
	defer closeClient("security-publicca", client.Close)

	calls := []callSpec{
		{
			name: "CreateExternalAccountKey",
			call: func(ctx context.Context) error {
				resp, err := client.CreateExternalAccountKey(ctx, &publiccapb.CreateExternalAccountKeyRequest{
					Parent: parent,
					ExternalAccountKey: &publiccapb.ExternalAccountKey{
						Name: externalAccountKeyName,
					},
				})
				if err == nil && resp != nil {
					logf("CreateExternalAccountKey name=%s keyId=%s", resp.GetName(), resp.GetKeyId())
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

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	return base.RoundTrip(clone)
}
