package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	websecurityscanner "cloud.google.com/go/websecurityscanner/apiv1"
	"cloud.google.com/go/websecurityscanner/apiv1/websecurityscannerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *websecurityscanner.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	scanConfigID := getenv("STACKYARD_GCP_SCAN_CONFIG_ID", "scan-config-1")
	scanRunID := getenv("STACKYARD_GCP_SCAN_RUN_ID", "scan-run-1")
	findingID := getenv("STACKYARD_GCP_FINDING_ID", "finding-1")

	parent := "projects/" + projectID
	scanConfigName := parent + "/scanConfigs/" + scanConfigID
	scanRunName := scanConfigName + "/scanRuns/" + scanRunID
	findingName := scanRunName + "/findings/" + findingID

	fmt.Printf("Stackyard GCP Web Security Scanner apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "websecurityscanner",
		},
	}

	client, err := websecurityscanner.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create websecurityscanner client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "CreateScanConfig",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				resp, err := c.CreateScanConfig(ctx, &websecurityscannerpb.CreateScanConfigRequest{
					Parent: parent,
					ScanConfig: &websecurityscannerpb.ScanConfig{
						DisplayName:  "Stackyard Scan Config",
						StartingUrls: []string{"https://scan-config-1.stackyard.test"},
						MaxQps:       15,
						UserAgent:    websecurityscannerpb.ScanConfig_CHROME_LINUX,
						RiskLevel:    websecurityscannerpb.ScanConfig_NORMAL,
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(resp.GetName()); name != "" {
					scanConfigName = name
					scanRunName = scanConfigName + "/scanRuns/" + scanRunID
					findingName = scanRunName + "/findings/" + findingID
				}
				return nil
			},
		},
		{
			name: "ListScanConfigs",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				it := c.ListScanConfigs(ctx, &websecurityscannerpb.ListScanConfigsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				item, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(item.GetName()); name != "" {
					scanConfigName = name
					scanRunName = scanConfigName + "/scanRuns/" + scanRunID
					findingName = scanRunName + "/findings/" + findingID
				}
				return nil
			},
		},
		{
			name: "GetScanConfig",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				_, err := c.GetScanConfig(ctx, &websecurityscannerpb.GetScanConfigRequest{
					Name: scanConfigName,
				})
				return err
			},
		},
		{
			name: "UpdateScanConfig",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				_, err := c.UpdateScanConfig(ctx, &websecurityscannerpb.UpdateScanConfigRequest{
					ScanConfig: &websecurityscannerpb.ScanConfig{
						Name:         scanConfigName,
						DisplayName:  "Stackyard Scan Config Updated",
						StartingUrls: []string{"https://scan-config-1.stackyard.test"},
						MaxQps:       10,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"display_name"},
					},
				})
				return err
			},
		},
		{
			name: "StartScanRun",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				resp, err := c.StartScanRun(ctx, &websecurityscannerpb.StartScanRunRequest{
					Name: scanConfigName,
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(resp.GetName()); name != "" {
					scanRunName = name
					findingName = scanRunName + "/findings/" + findingID
				}
				return nil
			},
		},
		{
			name: "GetScanRun",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				_, err := c.GetScanRun(ctx, &websecurityscannerpb.GetScanRunRequest{
					Name: scanRunName,
				})
				return err
			},
		},
		{
			name: "ListScanRuns",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				it := c.ListScanRuns(ctx, &websecurityscannerpb.ListScanRunsRequest{
					Parent:   scanConfigName,
					PageSize: 1,
				})
				item, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(item.GetName()); name != "" {
					scanRunName = name
					findingName = scanRunName + "/findings/" + findingID
				}
				return nil
			},
		},
		{
			name: "StopScanRun",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				_, err := c.StopScanRun(ctx, &websecurityscannerpb.StopScanRunRequest{
					Name: scanRunName,
				})
				return err
			},
		},
		{
			name: "ListCrawledUrls",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				it := c.ListCrawledUrls(ctx, &websecurityscannerpb.ListCrawledUrlsRequest{
					Parent:   scanRunName,
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
			name: "ListFindings",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				it := c.ListFindings(ctx, &websecurityscannerpb.ListFindingsRequest{
					Parent:   scanRunName,
					Filter:   "finding_type=MIXED_CONTENT",
					PageSize: 1,
				})
				item, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(item.GetName()); name != "" {
					findingName = name
				}
				return nil
			},
		},
		{
			name: "GetFinding",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				_, err := c.GetFinding(ctx, &websecurityscannerpb.GetFindingRequest{
					Name: findingName,
				})
				return err
			},
		},
		{
			name: "ListFindingTypeStats",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				_, err := c.ListFindingTypeStats(ctx, &websecurityscannerpb.ListFindingTypeStatsRequest{
					Parent: scanRunName,
				})
				return err
			},
		},
		{
			name: "DeleteScanConfig",
			call: func(ctx context.Context, c *websecurityscanner.Client) error {
				return c.DeleteScanConfig(ctx, &websecurityscannerpb.DeleteScanConfigRequest{
					Name: scanConfigName,
				})
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
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotImplemented {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close websecurityscanner client: %v\n", err)
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
