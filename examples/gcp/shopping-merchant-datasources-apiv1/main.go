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

	datasources "cloud.google.com/go/shopping/merchant/datasources/apiv1"
	"cloud.google.com/go/shopping/merchant/datasources/apiv1/datasourcespb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
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

	accountID := getenv("STACKYARD_GCP_MERCHANT_DATASOURCES_ACCOUNT_ID", "123456")
	displayName := getenv("STACKYARD_GCP_MERCHANT_DATASOURCES_DISPLAY_NAME", "Stackyard Created Data Source")
	feedLabel := strings.ToUpper(getenv("STACKYARD_GCP_MERCHANT_DATASOURCES_FEED_LABEL", "US"))
	contentLanguage := strings.ToLower(getenv("STACKYARD_GCP_MERCHANT_DATASOURCES_CONTENT_LANGUAGE", "en"))

	accountName := fmt.Sprintf("accounts/%s", accountID)
	dataSourceName := fmt.Sprintf("%s/dataSources/1001", accountName)

	fmt.Printf("Stackyard GCP Shopping Merchant Data Sources shopping/merchant/datasources/apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, accountName); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-datasources-apiv1",
		},
	}

	dataSourcesClient, err := datasources.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant datasources client: %v", err)
	}
	defer closeClient("merchant datasources", dataSourcesClient.Close)

	fileUploadsClient, err := datasources.NewFileUploadsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant file uploads client: %v", err)
	}
	defer closeClient("merchant file uploads", fileUploadsClient.Close)

	calls := []callSpec{
		{
			name: "CreateDataSource",
			call: func(ctx context.Context) error {
				resp, err := dataSourcesClient.CreateDataSource(ctx, &datasourcespb.CreateDataSourceRequest{
					Parent: accountName,
					DataSource: &datasourcespb.DataSource{
						DisplayName: displayName,
						Type: &datasourcespb.DataSource_PrimaryProductDataSource{
							PrimaryProductDataSource: &datasourcespb.PrimaryProductDataSource{
								FeedLabel:       proto.String(feedLabel),
								ContentLanguage: proto.String(contentLanguage),
								Countries:       []string{"US"},
							},
						},
						FileInput: &datasourcespb.FileInput{
							FileName: "products.csv",
						},
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(resp.GetName()); name != "" {
					dataSourceName = name
				}
				return nil
			},
		},
		{
			name: "ListDataSources",
			call: func(ctx context.Context) error {
				it := dataSourcesClient.ListDataSources(ctx, &datasourcespb.ListDataSourcesRequest{
					Parent:   accountName,
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
			name: "GetDataSource",
			call: func(ctx context.Context) error {
				_, err := dataSourcesClient.GetDataSource(ctx, &datasourcespb.GetDataSourceRequest{Name: dataSourceName})
				return err
			},
		},
		{
			name: "UpdateDataSource",
			call: func(ctx context.Context) error {
				_, err := dataSourcesClient.UpdateDataSource(ctx, &datasourcespb.UpdateDataSourceRequest{
					DataSource: &datasourcespb.DataSource{
						Name:        dataSourceName,
						DisplayName: displayName + " Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "FetchDataSource",
			call: func(ctx context.Context) error {
				return dataSourcesClient.FetchDataSource(ctx, &datasourcespb.FetchDataSourceRequest{Name: dataSourceName})
			},
		},
		{
			name: "GetFileUpload",
			call: func(ctx context.Context) error {
				_, err := fileUploadsClient.GetFileUpload(ctx, &datasourcespb.GetFileUploadRequest{
					Name: dataSourceName + "/fileUploads/latest",
				})
				return err
			},
		},
		{
			name: "DeleteDataSource",
			call: func(ctx context.Context) error {
				return dataSourcesClient.DeleteDataSource(ctx, &datasourcespb.DeleteDataSourceRequest{Name: dataSourceName})
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
	target := strings.TrimRight(endpoint, "/") + "/datasources/v1/" + accountName + "/dataSources?pageSize=1"
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

func getenvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
