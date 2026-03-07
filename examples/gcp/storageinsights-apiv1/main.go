package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	storageinsights "cloud.google.com/go/storageinsights/apiv1"
	"cloud.google.com/go/storageinsights/apiv1/storageinsightspb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *storageinsights.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION_ID", "us-central1")
	reportConfigID := getenv("STACKYARD_GCP_STORAGE_INSIGHTS_REPORT_CONFIG_ID", "reportconfig1")
	datasetConfigID := getenv("STACKYARD_GCP_STORAGE_INSIGHTS_DATASET_CONFIG_ID", "datasetconfig1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	reportConfigName := fmt.Sprintf("%s/reportConfigs/%s", parent, reportConfigID)
	reportDetailName := fmt.Sprintf("%s/reportDetails/reportdetail1", reportConfigName)
	datasetConfigName := fmt.Sprintf("%s/datasetConfigs/%s", parent, datasetConfigID)
	operationName := fmt.Sprintf("%s/operations/createDatasetConfig.%s", parent, datasetConfigID)

	fmt.Printf("Stackyard GCP Storage Insights apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "storageinsights",
		},
	}

	client, err := storageinsights.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create storageinsights client: %v", err)
	}
	defer closeClient(client.Close)

	reportConfig := func(displayName string) *storageinsightspb.ReportConfig {
		return &storageinsightspb.ReportConfig{
			Name:        reportConfigName,
			DisplayName: displayName,
			FrequencyOptions: &storageinsightspb.FrequencyOptions{
				Frequency: storageinsightspb.FrequencyOptions_DAILY,
			},
			ReportFormat: &storageinsightspb.ReportConfig_CsvOptions{
				CsvOptions: &storageinsightspb.CSVOptions{
					RecordSeparator: "\n",
					Delimiter:       ",",
					HeaderRequired:  true,
				},
			},
			ReportKind: &storageinsightspb.ReportConfig_ObjectMetadataReportOptions{
				ObjectMetadataReportOptions: &storageinsightspb.ObjectMetadataReportOptions{
					MetadataFields: []string{"name", "size", "updated"},
				},
			},
			Labels: map[string]string{
				"env": "staged",
			},
		}
	}
	datasetConfig := func(description string) *storageinsightspb.DatasetConfig {
		return &storageinsightspb.DatasetConfig{
			Name:        datasetConfigName,
			Description: description,
			SourceOptions: &storageinsightspb.DatasetConfig_SourceProjects_{
				SourceProjects: &storageinsightspb.DatasetConfig_SourceProjects{
					ProjectNumbers: []int64{123456789},
				},
			},
		}
	}

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				location, err := it.Next()
				if err != nil {
					return err
				}
				if strings.TrimSpace(location.GetName()) == "" {
					return fmt.Errorf("list locations returned empty location name")
				}
				return nil
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				location, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				if err != nil {
					return err
				}
				if strings.TrimSpace(location.GetName()) == "" {
					return fmt.Errorf("get location returned empty name")
				}
				return nil
			},
		},
		{
			name: "ListReportConfigs",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				it := c.ListReportConfigs(ctx, &storageinsightspb.ListReportConfigsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				item, err := it.Next()
				if err != nil {
					return err
				}
				if strings.TrimSpace(item.GetName()) == "" {
					return fmt.Errorf("list report configs returned empty name")
				}
				return nil
			},
		},
		{
			name: "CreateReportConfig",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				created, err := c.CreateReportConfig(ctx, &storageinsightspb.CreateReportConfigRequest{
					Parent:       parent,
					RequestId:    "11111111-1111-4111-8111-111111111111",
					ReportConfig: reportConfig("Stackyard Report Config"),
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(created.GetName()) == "" {
					return fmt.Errorf("create report config returned empty name")
				}
				return nil
			},
		},
		{
			name: "GetReportConfig",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				got, err := c.GetReportConfig(ctx, &storageinsightspb.GetReportConfigRequest{Name: reportConfigName})
				if err != nil {
					return err
				}
				if strings.TrimSpace(got.GetName()) == "" {
					return fmt.Errorf("get report config returned empty name")
				}
				return nil
			},
		},
		{
			name: "UpdateReportConfig",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				updated, err := c.UpdateReportConfig(ctx, &storageinsightspb.UpdateReportConfigRequest{
					RequestId:    "22222222-2222-4222-8222-222222222222",
					ReportConfig: reportConfig("Stackyard Updated Report Config"),
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"display_name"},
					},
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(updated.GetName()) == "" {
					return fmt.Errorf("update report config returned empty name")
				}
				return nil
			},
		},
		{
			name: "ListReportDetails",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				it := c.ListReportDetails(ctx, &storageinsightspb.ListReportDetailsRequest{
					Parent:   reportConfigName,
					PageSize: 1,
				})
				item, err := it.Next()
				if err != nil {
					return err
				}
				if strings.TrimSpace(item.GetName()) == "" {
					return fmt.Errorf("list report details returned empty name")
				}
				return nil
			},
		},
		{
			name: "GetReportDetail",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				got, err := c.GetReportDetail(ctx, &storageinsightspb.GetReportDetailRequest{Name: reportDetailName})
				if err != nil {
					return err
				}
				if strings.TrimSpace(got.GetName()) == "" {
					return fmt.Errorf("get report detail returned empty name")
				}
				return nil
			},
		},
		{
			name: "ListDatasetConfigs",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				it := c.ListDatasetConfigs(ctx, &storageinsightspb.ListDatasetConfigsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				item, err := it.Next()
				if err != nil {
					return err
				}
				if strings.TrimSpace(item.GetName()) == "" {
					return fmt.Errorf("list dataset configs returned empty name")
				}
				return nil
			},
		},
		{
			name: "CreateDatasetConfig",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				op, err := c.CreateDatasetConfig(ctx, &storageinsightspb.CreateDatasetConfigRequest{
					Parent:          parent,
					DatasetConfigId: datasetConfigID,
					RequestId:       "33333333-3333-4333-8333-333333333333",
					DatasetConfig:   datasetConfig("Stackyard dataset config"),
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				created, err := op.Wait(ctx)
				if err != nil {
					return err
				}
				if strings.TrimSpace(created.GetName()) == "" {
					return fmt.Errorf("create dataset config operation returned empty dataset config name")
				}
				return nil
			},
		},
		{
			name: "GetDatasetConfig",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				got, err := c.GetDatasetConfig(ctx, &storageinsightspb.GetDatasetConfigRequest{Name: datasetConfigName})
				if err != nil {
					return err
				}
				if strings.TrimSpace(got.GetName()) == "" {
					return fmt.Errorf("get dataset config returned empty name")
				}
				return nil
			},
		},
		{
			name: "LinkDataset",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				op, err := c.LinkDataset(ctx, &storageinsightspb.LinkDatasetRequest{Name: datasetConfigName})
				if err != nil {
					return err
				}
				if _, err := op.Wait(ctx); err != nil {
					return err
				}
				return nil
			},
		},
		{
			name: "UnlinkDataset",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				op, err := c.UnlinkDataset(ctx, &storageinsightspb.UnlinkDatasetRequest{Name: datasetConfigName})
				if err != nil {
					return err
				}
				return op.Wait(ctx)
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				got, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				if err != nil {
					return err
				}
				if strings.TrimSpace(got.GetName()) == "" {
					return fmt.Errorf("get operation returned empty name")
				}
				return nil
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
					PageSize: 1,
				})
				item, err := it.Next()
				if err != nil {
					return err
				}
				if strings.TrimSpace(item.GetName()) == "" {
					return fmt.Errorf("list operations returned empty operation name")
				}
				return nil
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteDatasetConfig",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				op, err := c.DeleteDatasetConfig(ctx, &storageinsightspb.DeleteDatasetConfigRequest{
					Name:      datasetConfigName,
					RequestId: "44444444-4444-4444-8444-444444444444",
				})
				if err != nil {
					return err
				}
				return op.Wait(ctx)
			},
		},
		{
			name: "DeleteReportConfig",
			call: func(ctx context.Context, c *storageinsights.Client) error {
				return c.DeleteReportConfig(ctx, &storageinsightspb.DeleteReportConfigRequest{
					Name:      reportConfigName,
					Force:     true,
					RequestId: "55555555-5555-4555-8555-555555555555",
				})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
		if err == nil {
			logf("%s succeeded", call.name)
			continue
		}
		if err == iterator.Done {
			logf("%s returned no items", call.name)
			continue
		}
		exitf("%s failed: %v", call.name, err)
	}

	fmt.Println("Done.")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close storageinsights client: %v\n", err)
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
