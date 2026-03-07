package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	cloudquotas "cloud.google.com/go/cloudquotas/apiv1"
	"cloud.google.com/go/cloudquotas/apiv1/cloudquotaspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *cloudquotas.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	location := getenv("STACKYARD_GCP_LOCATION", "global")
	service := getenv("STACKYARD_GCP_QUOTA_SERVICE", "compute.googleapis.com")
	quotaID := getenv("STACKYARD_GCP_QUOTA_ID", "CpusPerProjectPerRegion")
	quotaPreferenceID := getenv("STACKYARD_GCP_QUOTA_PREFERENCE_ID", "team-config")

	quotaInfosParent := fmt.Sprintf("projects/%s/locations/%s/services/%s", projectID, location, service)
	quotaPreferencesParent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)
	quotaInfoName := quotaInfosParent + "/quotaInfos/" + quotaID
	quotaPreferenceName := quotaPreferencesParent + "/quotaPreferences/" + quotaPreferenceID

	fmt.Printf("Stackyard GCP Cloud Quotas apiv1 client using %s\n", apiEndpoint)

	client, err := cloudquotas.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloudquotas client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListQuotaInfos",
			call: func(ctx context.Context, c *cloudquotas.Client) error {
				it := c.ListQuotaInfos(ctx, &cloudquotaspb.ListQuotaInfosRequest{
					Parent:   quotaInfosParent,
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
			name: "GetQuotaInfo",
			call: func(ctx context.Context, c *cloudquotas.Client) error {
				_, err := c.GetQuotaInfo(ctx, &cloudquotaspb.GetQuotaInfoRequest{
					Name: quotaInfoName,
				})
				return err
			},
		},
		{
			name: "ListQuotaPreferences",
			call: func(ctx context.Context, c *cloudquotas.Client) error {
				it := c.ListQuotaPreferences(ctx, &cloudquotaspb.ListQuotaPreferencesRequest{
					Parent:   quotaPreferencesParent,
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
			name: "GetQuotaPreference",
			call: func(ctx context.Context, c *cloudquotas.Client) error {
				_, err := c.GetQuotaPreference(ctx, &cloudquotaspb.GetQuotaPreferenceRequest{
					Name: quotaPreferenceName,
				})
				return err
			},
		},
		{
			name: "CreateQuotaPreference",
			call: func(ctx context.Context, c *cloudquotas.Client) error {
				_, err := c.CreateQuotaPreference(ctx, &cloudquotaspb.CreateQuotaPreferenceRequest{
					Parent:            quotaPreferencesParent,
					QuotaPreferenceId: quotaPreferenceID,
					QuotaPreference: &cloudquotaspb.QuotaPreference{
						Service: service,
						QuotaId: quotaID,
						QuotaConfig: &cloudquotaspb.QuotaConfig{
							PreferredValue: 16,
						},
						Dimensions: map[string]string{
							"region": "us-central1",
						},
						Justification: "stackyard staged example",
						ContactEmail:  "stackyard@example.com",
					},
				})
				return err
			},
		},
		{
			name: "UpdateQuotaPreference",
			call: func(ctx context.Context, c *cloudquotas.Client) error {
				_, err := c.UpdateQuotaPreference(ctx, &cloudquotaspb.UpdateQuotaPreferenceRequest{
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"quota_config.preferred_value", "justification"},
					},
					QuotaPreference: &cloudquotaspb.QuotaPreference{
						Name:    quotaPreferenceName,
						Service: service,
						QuotaId: quotaID,
						QuotaConfig: &cloudquotaspb.QuotaConfig{
							PreferredValue: 32,
						},
						Justification: "stackyard staged update",
					},
				})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
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

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close cloudquotas client: %v\n", err)
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
