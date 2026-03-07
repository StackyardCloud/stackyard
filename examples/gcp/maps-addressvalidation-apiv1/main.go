package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	addressvalidation "cloud.google.com/go/maps/addressvalidation/apiv1"
	"cloud.google.com/go/maps/addressvalidation/apiv1/addressvalidationpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	postaladdress "google.golang.org/genproto/googleapis/type/postaladdress"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *addressvalidation.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	responseID := getenv("STACKYARD_GCP_ADDRESSVALIDATION_RESPONSE_ID", "stackyard-response-id")
	feedbackConclusion := addressvalidationpb.ProvideValidationFeedbackRequest_VALIDATED_VERSION_USED

	fmt.Printf("Stackyard GCP Maps Address Validation apiv1 client using %s\n", apiEndpoint)

	client, err := addressvalidation.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create maps addressvalidation client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ValidateAddress",
			call: func(ctx context.Context, c *addressvalidation.Client) error {
				resp, err := c.ValidateAddress(ctx, &addressvalidationpb.ValidateAddressRequest{
					Address: &postaladdress.PostalAddress{
						RegionCode:         "US",
						AddressLines:       []string{"1600 Amphitheatre Parkway"},
						Locality:           "Mountain View",
						AdministrativeArea: "CA",
						PostalCode:         "94043",
					},
					EnableUspsCass: true,
				})
				if err == nil && strings.TrimSpace(resp.GetResponseId()) != "" {
					responseID = resp.GetResponseId()
				}
				return err
			},
		},
		{
			name: "ProvideValidationFeedback",
			call: func(ctx context.Context, c *addressvalidation.Client) error {
				_, err := c.ProvideValidationFeedback(ctx, &addressvalidationpb.ProvideValidationFeedbackRequest{
					Conclusion: feedbackConclusion,
					ResponseId: responseID,
				})
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
		fmt.Fprintf(os.Stderr, "warning: close maps addressvalidation client: %v\n", err)
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
