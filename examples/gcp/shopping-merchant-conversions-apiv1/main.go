package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	conversions "cloud.google.com/go/shopping/merchant/conversions/apiv1"
	"cloud.google.com/go/shopping/merchant/conversions/apiv1/conversionspb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	accountID := getenv("STACKYARD_GCP_MERCHANT_CONVERSIONS_ACCOUNT_ID", "123456")
	gaPropertyID := getenvInt64("STACKYARD_GCP_MERCHANT_CONVERSIONS_GA_PROPERTY_ID", 2001)
	displayName := getenv("STACKYARD_GCP_MERCHANT_CONVERSIONS_DISPLAY_NAME", "Primary Destination")
	currencyCode := strings.ToUpper(getenv("STACKYARD_GCP_MERCHANT_CONVERSIONS_CURRENCY", "USD"))

	accountName := fmt.Sprintf("accounts/%s", accountID)

	fmt.Printf("Stackyard GCP Shopping Merchant Conversions shopping/merchant/conversions/apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, accountName); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-conversions-apiv1",
		},
	}

	client, err := conversions.NewConversionSourcesRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant conversions client: %v", err)
	}
	defer closeClient("merchant conversions", client.Close)

	mcdName := fmt.Sprintf("%s/conversionSources/mcdn:1001", accountName)
	gaName := fmt.Sprintf("%s/conversionSources/galk:%d", accountName, gaPropertyID)

	calls := []callSpec{
		{
			name: "CreateConversionSourceMCD",
			call: func(ctx context.Context) error {
				resp, err := client.CreateConversionSource(ctx, &conversionspb.CreateConversionSourceRequest{
					Parent: accountName,
					ConversionSource: &conversionspb.ConversionSource{
						SourceData: &conversionspb.ConversionSource_MerchantCenterDestination{
							MerchantCenterDestination: &conversionspb.MerchantCenterDestination{
								DisplayName:  displayName,
								CurrencyCode: currencyCode,
								AttributionSettings: &conversionspb.AttributionSettings{
									AttributionLookbackWindowDays: 30,
									AttributionModel:              conversionspb.AttributionSettings_CROSS_CHANNEL_LAST_CLICK,
								},
							},
						},
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(resp.GetName()); name != "" {
					mcdName = name
				}
				return nil
			},
		},
		{
			name: "CreateConversionSourceGALink",
			call: func(ctx context.Context) error {
				resp, err := client.CreateConversionSource(ctx, &conversionspb.CreateConversionSourceRequest{
					Parent: accountName,
					ConversionSource: &conversionspb.ConversionSource{
						SourceData: &conversionspb.ConversionSource_GoogleAnalyticsLink{
							GoogleAnalyticsLink: &conversionspb.GoogleAnalyticsLink{
								PropertyId: gaPropertyID,
							},
						},
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(resp.GetName()); name != "" {
					gaName = name
				}
				return nil
			},
		},
		{
			name: "ListConversionSources",
			call: func(ctx context.Context) error {
				it := client.ListConversionSources(ctx, &conversionspb.ListConversionSourcesRequest{Parent: accountName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetConversionSource",
			call: func(ctx context.Context) error {
				_, err := client.GetConversionSource(ctx, &conversionspb.GetConversionSourceRequest{Name: mcdName})
				return err
			},
		},
		{
			name: "UpdateConversionSource",
			call: func(ctx context.Context) error {
				_, err := client.UpdateConversionSource(ctx, &conversionspb.UpdateConversionSourceRequest{
					ConversionSource: &conversionspb.ConversionSource{
						Name: mcdName,
						SourceData: &conversionspb.ConversionSource_MerchantCenterDestination{
							MerchantCenterDestination: &conversionspb.MerchantCenterDestination{
								DisplayName: displayName + " Updated",
							},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"merchant_center_destination.display_name"}},
				})
				return err
			},
		},
		{
			name: "DeleteConversionSourceMCD",
			call: func(ctx context.Context) error {
				return client.DeleteConversionSource(ctx, &conversionspb.DeleteConversionSourceRequest{Name: mcdName})
			},
		},
		{
			name: "UndeleteConversionSourceMCD",
			call: func(ctx context.Context) error {
				_, err := client.UndeleteConversionSource(ctx, &conversionspb.UndeleteConversionSourceRequest{Name: mcdName})
				return err
			},
		},
		{
			name: "DeleteConversionSourceGALink",
			call: func(ctx context.Context) error {
				return client.DeleteConversionSource(ctx, &conversionspb.DeleteConversionSourceRequest{Name: gaName})
			},
		},
		{
			name: "ListConversionSourcesShowDeleted",
			call: func(ctx context.Context) error {
				it := client.ListConversionSources(ctx, &conversionspb.ListConversionSourcesRequest{
					Parent:      accountName,
					PageSize:    1,
					ShowDeleted: true,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
	}

	for _, spec := range calls {
		if err := spec.call(ctx); err != nil {
			exitf("%s failed: %v", spec.name, err)
		}
		fmt.Printf("%s succeeded\n", spec.name)
	}

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, endpoint, accountName string) error {
	target := strings.TrimRight(endpoint, "/") + "/conversions/v1/" + accountName + "/conversionSources?pageSize=1"
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < http.StatusInternalServerError {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for stackyard at %s", target)
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
	if strings.TrimSpace(t.serviceName) != "" {
		clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	}
	return base.RoundTrip(clone)
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to close %s client: %v\n", name, err)
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
