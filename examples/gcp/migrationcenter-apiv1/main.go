package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	migrationcenter "cloud.google.com/go/migrationcenter/apiv1"
	"cloud.google.com/go/migrationcenter/apiv1/migrationcenterpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *migrationcenter.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	assetID := getenv("STACKYARD_GCP_MIGRATIONCENTER_ASSET_ID", "asset-a")
	sourceID := getenv("STACKYARD_GCP_MIGRATIONCENTER_SOURCE_ID", "source-a")
	importJobID := getenv("STACKYARD_GCP_MIGRATIONCENTER_IMPORT_JOB_ID", "job-a")
	importDataFileID := getenv("STACKYARD_GCP_MIGRATIONCENTER_IMPORT_DATA_FILE_ID", "file-a")
	groupID := getenv("STACKYARD_GCP_MIGRATIONCENTER_GROUP_ID", "group-a")
	preferenceSetID := getenv("STACKYARD_GCP_MIGRATIONCENTER_PREFERENCE_SET_ID", "prefs-a")
	reportConfigID := getenv("STACKYARD_GCP_MIGRATIONCENTER_REPORT_CONFIG_ID", "report-config-a")
	reportID := getenv("STACKYARD_GCP_MIGRATIONCENTER_REPORT_ID", "report-a")
	errorFrameID := getenv("STACKYARD_GCP_MIGRATIONCENTER_ERROR_FRAME_ID", "frame-a")
	operationID := getenv("STACKYARD_GCP_MIGRATIONCENTER_OPERATION_ID", "op-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	assetName := fmt.Sprintf("%s/assets/%s", locationName, assetID)
	sourceName := fmt.Sprintf("%s/sources/%s", locationName, sourceID)
	importJobName := fmt.Sprintf("%s/importJobs/%s", locationName, importJobID)
	importDataFileName := fmt.Sprintf("%s/importDataFiles/%s", importJobName, importDataFileID)
	groupName := fmt.Sprintf("%s/groups/%s", locationName, groupID)
	preferenceSetName := fmt.Sprintf("%s/preferenceSets/%s", locationName, preferenceSetID)
	settingsName := fmt.Sprintf("%s/settings", locationName)
	reportConfigName := fmt.Sprintf("%s/reportConfigs/%s", locationName, reportConfigID)
	reportName := fmt.Sprintf("%s/reports/%s", reportConfigName, reportID)
	errorFrameName := fmt.Sprintf("%s/errorFrames/%s", sourceName, errorFrameID)
	operationName := fmt.Sprintf("%s/operations/%s", locationName, operationID)

	fmt.Printf("Stackyard GCP Migration Center apiv1 client using %s\n", apiEndpoint)

	client, err := migrationcenter.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create migration center client: %v", err)
	}
	defer closeClient("migration center", client.Close)

	calls := []callSpec{
		{
			name: "ListAssets",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				it := c.ListAssets(ctx, &migrationcenterpb.ListAssetsRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetAsset",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetAsset(ctx, &migrationcenterpb.GetAssetRequest{Name: assetName})
				return err
			},
		},
		{
			name: "UpdateAsset",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.UpdateAsset(ctx, &migrationcenterpb.UpdateAssetRequest{
					Asset: &migrationcenterpb.Asset{
						Name:   assetName,
						Labels: map[string]string{"env": "local"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "DeleteAsset",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				return c.DeleteAsset(ctx, &migrationcenterpb.DeleteAssetRequest{Name: assetName})
			},
		},
		{
			name: "BatchUpdateAssets",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.BatchUpdateAssets(ctx, &migrationcenterpb.BatchUpdateAssetsRequest{
					Parent: locationName,
					Requests: []*migrationcenterpb.UpdateAssetRequest{
						{
							Asset: &migrationcenterpb.Asset{
								Name:       assetName,
								Attributes: map[string]string{"owner": "stackyard"},
							},
							UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"attributes"}},
						},
					},
				})
				return err
			},
		},
		{
			name: "BatchDeleteAssets",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				return c.BatchDeleteAssets(ctx, &migrationcenterpb.BatchDeleteAssetsRequest{
					Parent:       locationName,
					Names:        []string{assetName},
					AllowMissing: true,
				})
			},
		},
		{
			name: "ReportAssetFrames",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.ReportAssetFrames(ctx, &migrationcenterpb.ReportAssetFramesRequest{
					Parent: locationName,
					Source: sourceName,
					Frames: &migrationcenterpb.Frames{},
				})
				return err
			},
		},
		{
			name: "AggregateAssetsValues",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.AggregateAssetsValues(ctx, &migrationcenterpb.AggregateAssetsValuesRequest{
					Parent: locationName,
				})
				return err
			},
		},
		{
			name: "ListImportJobs",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				it := c.ListImportJobs(ctx, &migrationcenterpb.ListImportJobsRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetImportJob",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetImportJob(ctx, &migrationcenterpb.GetImportJobRequest{Name: importJobName})
				return err
			},
		},
		{
			name: "CreateImportJob",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.CreateImportJob(ctx, &migrationcenterpb.CreateImportJobRequest{
					Parent:      locationName,
					ImportJobId: importJobID,
					ImportJob: &migrationcenterpb.ImportJob{
						Name:        importJobName,
						DisplayName: "stackyard import",
						AssetSource: sourceName,
					},
				})
				return err
			},
		},
		{
			name: "UpdateImportJob",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.UpdateImportJob(ctx, &migrationcenterpb.UpdateImportJobRequest{
					ImportJob: &migrationcenterpb.ImportJob{
						Name:        importJobName,
						DisplayName: "stackyard import updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "ValidateImportJob",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.ValidateImportJob(ctx, &migrationcenterpb.ValidateImportJobRequest{Name: importJobName})
				return err
			},
		},
		{
			name: "RunImportJob",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.RunImportJob(ctx, &migrationcenterpb.RunImportJobRequest{Name: importJobName})
				return err
			},
		},
		{
			name: "DeleteImportJob",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.DeleteImportJob(ctx, &migrationcenterpb.DeleteImportJobRequest{Name: importJobName})
				return err
			},
		},
		{
			name: "ListImportDataFiles",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				it := c.ListImportDataFiles(ctx, &migrationcenterpb.ListImportDataFilesRequest{
					Parent:   importJobName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CreateImportDataFile",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.CreateImportDataFile(ctx, &migrationcenterpb.CreateImportDataFileRequest{
					Parent:           importJobName,
					ImportDataFileId: importDataFileID,
					ImportDataFile: &migrationcenterpb.ImportDataFile{
						Name:        importDataFileName,
						DisplayName: "stackyard source file",
						Format:      migrationcenterpb.ImportJobFormat_IMPORT_JOB_FORMAT_RVTOOLS_CSV,
					},
				})
				return err
			},
		},
		{
			name: "GetImportDataFile",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetImportDataFile(ctx, &migrationcenterpb.GetImportDataFileRequest{Name: importDataFileName})
				return err
			},
		},
		{
			name: "DeleteImportDataFile",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.DeleteImportDataFile(ctx, &migrationcenterpb.DeleteImportDataFileRequest{Name: importDataFileName})
				return err
			},
		},
		{
			name: "ListGroups",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				it := c.ListGroups(ctx, &migrationcenterpb.ListGroupsRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CreateGroup",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.CreateGroup(ctx, &migrationcenterpb.CreateGroupRequest{
					Parent:  locationName,
					GroupId: groupID,
					Group: &migrationcenterpb.Group{
						Name:        groupName,
						DisplayName: "stackyard group",
					},
				})
				return err
			},
		},
		{
			name: "GetGroup",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetGroup(ctx, &migrationcenterpb.GetGroupRequest{Name: groupName})
				return err
			},
		},
		{
			name: "UpdateGroup",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.UpdateGroup(ctx, &migrationcenterpb.UpdateGroupRequest{
					Group: &migrationcenterpb.Group{
						Name:        groupName,
						Description: "updated by stackyard example",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "AddAssetsToGroup",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.AddAssetsToGroup(ctx, &migrationcenterpb.AddAssetsToGroupRequest{
					Group: groupName,
					Assets: &migrationcenterpb.AssetList{
						AssetIds: []string{assetName},
					},
					AllowExisting: true,
				})
				return err
			},
		},
		{
			name: "RemoveAssetsFromGroup",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.RemoveAssetsFromGroup(ctx, &migrationcenterpb.RemoveAssetsFromGroupRequest{
					Group: groupName,
					Assets: &migrationcenterpb.AssetList{
						AssetIds: []string{assetName},
					},
					AllowMissing: true,
				})
				return err
			},
		},
		{
			name: "DeleteGroup",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.DeleteGroup(ctx, &migrationcenterpb.DeleteGroupRequest{Name: groupName})
				return err
			},
		},
		{
			name: "ListErrorFrames",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				it := c.ListErrorFrames(ctx, &migrationcenterpb.ListErrorFramesRequest{
					Parent:   sourceName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetErrorFrame",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetErrorFrame(ctx, &migrationcenterpb.GetErrorFrameRequest{Name: errorFrameName})
				return err
			},
		},
		{
			name: "ListSources",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				it := c.ListSources(ctx, &migrationcenterpb.ListSourcesRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CreateSource",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.CreateSource(ctx, &migrationcenterpb.CreateSourceRequest{
					Parent:   locationName,
					SourceId: sourceID,
					Source: &migrationcenterpb.Source{
						Name:        sourceName,
						DisplayName: "stackyard source",
						Description: "local emulated source",
						Type:        migrationcenterpb.Source_SOURCE_TYPE_CUSTOM,
						Priority:    1,
					},
				})
				return err
			},
		},
		{
			name: "GetSource",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetSource(ctx, &migrationcenterpb.GetSourceRequest{Name: sourceName})
				return err
			},
		},
		{
			name: "UpdateSource",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.UpdateSource(ctx, &migrationcenterpb.UpdateSourceRequest{
					Source: &migrationcenterpb.Source{
						Name:        sourceName,
						Description: "updated by stackyard example",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "DeleteSource",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.DeleteSource(ctx, &migrationcenterpb.DeleteSourceRequest{Name: sourceName})
				return err
			},
		},
		{
			name: "ListPreferenceSets",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				it := c.ListPreferenceSets(ctx, &migrationcenterpb.ListPreferenceSetsRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CreatePreferenceSet",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.CreatePreferenceSet(ctx, &migrationcenterpb.CreatePreferenceSetRequest{
					Parent:          locationName,
					PreferenceSetId: preferenceSetID,
					PreferenceSet: &migrationcenterpb.PreferenceSet{
						Name:        preferenceSetName,
						DisplayName: "stackyard defaults",
					},
				})
				return err
			},
		},
		{
			name: "GetPreferenceSet",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetPreferenceSet(ctx, &migrationcenterpb.GetPreferenceSetRequest{Name: preferenceSetName})
				return err
			},
		},
		{
			name: "UpdatePreferenceSet",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.UpdatePreferenceSet(ctx, &migrationcenterpb.UpdatePreferenceSetRequest{
					PreferenceSet: &migrationcenterpb.PreferenceSet{
						Name:        preferenceSetName,
						Description: "updated by stackyard example",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "DeletePreferenceSet",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.DeletePreferenceSet(ctx, &migrationcenterpb.DeletePreferenceSetRequest{Name: preferenceSetName})
				return err
			},
		},
		{
			name: "GetSettings",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetSettings(ctx, &migrationcenterpb.GetSettingsRequest{Name: settingsName})
				return err
			},
		},
		{
			name: "UpdateSettings",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.UpdateSettings(ctx, &migrationcenterpb.UpdateSettingsRequest{
					Settings: &migrationcenterpb.Settings{
						Name:          settingsName,
						PreferenceSet: preferenceSetName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"preference_set"}},
				})
				return err
			},
		},
		{
			name: "ListReportConfigs",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				it := c.ListReportConfigs(ctx, &migrationcenterpb.ListReportConfigsRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CreateReportConfig",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.CreateReportConfig(ctx, &migrationcenterpb.CreateReportConfigRequest{
					Parent:         locationName,
					ReportConfigId: reportConfigID,
					ReportConfig: &migrationcenterpb.ReportConfig{
						Name:        reportConfigName,
						DisplayName: "stackyard tco config",
						GroupPreferencesetAssignments: []*migrationcenterpb.ReportConfig_GroupPreferenceSetAssignment{
							{
								Group:         groupName,
								PreferenceSet: preferenceSetName,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetReportConfig",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetReportConfig(ctx, &migrationcenterpb.GetReportConfigRequest{Name: reportConfigName})
				return err
			},
		},
		{
			name: "DeleteReportConfig",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.DeleteReportConfig(ctx, &migrationcenterpb.DeleteReportConfigRequest{Name: reportConfigName})
				return err
			},
		},
		{
			name: "ListReports",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				it := c.ListReports(ctx, &migrationcenterpb.ListReportsRequest{
					Parent:   reportConfigName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CreateReport",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.CreateReport(ctx, &migrationcenterpb.CreateReportRequest{
					Parent:   reportConfigName,
					ReportId: reportID,
					Report: &migrationcenterpb.Report{
						Name:        reportName,
						DisplayName: "stackyard tco report",
						Type:        migrationcenterpb.Report_TOTAL_COST_OF_OWNERSHIP,
					},
				})
				return err
			},
		},
		{
			name: "GetReport",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetReport(ctx, &migrationcenterpb.GetReportRequest{Name: reportName})
				return err
			},
		},
		{
			name: "DeleteReport",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.DeleteReport(ctx, &migrationcenterpb.DeleteReportRequest{Name: reportName})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *migrationcenter.Client) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
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

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
