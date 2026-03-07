package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	financialservices "cloud.google.com/go/financialservices/apiv1"
	"cloud.google.com/go/financialservices/apiv1/financialservicespb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *financialservices.AMLClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_FINANCIALSERVICES_LOCATION", "us-central1")
	instanceID := getenv("STACKYARD_GCP_FINANCIALSERVICES_INSTANCE_ID", "aml-instance")
	datasetID := getenv("STACKYARD_GCP_FINANCIALSERVICES_DATASET_ID", "dataset-1")
	modelID := getenv("STACKYARD_GCP_FINANCIALSERVICES_MODEL_ID", "model-1")
	engineConfigID := getenv("STACKYARD_GCP_FINANCIALSERVICES_ENGINE_CONFIG_ID", "engine-config-1")
	engineVersionID := getenv("STACKYARD_GCP_FINANCIALSERVICES_ENGINE_VERSION_ID", "engine-version-1")
	predictionResultID := getenv("STACKYARD_GCP_FINANCIALSERVICES_PREDICTION_RESULT_ID", "prediction-1")
	backtestResultID := getenv("STACKYARD_GCP_FINANCIALSERVICES_BACKTEST_RESULT_ID", "backtest-1")
	operationID := getenv("STACKYARD_GCP_FINANCIALSERVICES_OPERATION_ID", "op-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := projectName + "/locations/" + locationID
	instanceName := locationName + "/instances/" + instanceID
	datasetName := instanceName + "/datasets/" + datasetID
	modelName := instanceName + "/models/" + modelID
	engineConfigName := instanceName + "/engineConfigs/" + engineConfigID
	engineVersionName := instanceName + "/engineVersions/" + engineVersionID
	predictionResultName := instanceName + "/predictionResults/" + predictionResultID
	backtestResultName := instanceName + "/backtestResults/" + backtestResultID
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Financial Services apiv1 client using %s\n", apiEndpoint)

	client, err := financialservices.NewAMLRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create financial services client: %v", err)
	}
	defer closeClient(client.Close)

	registeredPartiesDest := &financialservicespb.BigQueryDestination{
		TableUri:         fmt.Sprintf("bq://%s.stackyard.registered_parties", projectID),
		WriteDisposition: financialservicespb.BigQueryDestination_WRITE_DISPOSITION_UNSPECIFIED,
	}
	metadataDest := &financialservicespb.BigQueryDestination{
		TableUri:         fmt.Sprintf("bq://%s.stackyard.financialservices_metadata", projectID),
		WriteDisposition: financialservicespb.BigQueryDestination_WRITE_DISPOSITION_UNSPECIFIED,
	}

	calls := []callSpec{
		{
			name: "ListInstances",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				it := c.ListInstances(ctx, &financialservicespb.ListInstancesRequest{
					Parent:   locationName,
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
			name: "GetInstance",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.GetInstance(ctx, &financialservicespb.GetInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "CreateInstance",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.CreateInstance(ctx, &financialservicespb.CreateInstanceRequest{
					Parent:     locationName,
					InstanceId: instanceID,
					Instance:   &financialservicespb.Instance{Name: instanceName},
				})
				return err
			},
		},
		{
			name: "UpdateInstance",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.UpdateInstance(ctx, &financialservicespb.UpdateInstanceRequest{
					Instance: &financialservicespb.Instance{Name: instanceName},
				})
				return err
			},
		},
		{
			name: "DeleteInstance",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.DeleteInstance(ctx, &financialservicespb.DeleteInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "ImportRegisteredParties",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.ImportRegisteredParties(ctx, &financialservicespb.ImportRegisteredPartiesRequest{
					Name:           instanceName,
					PartyTables:    []string{fmt.Sprintf("bq://%s.stackyard.party_table", projectID)},
					Mode:           financialservicespb.ImportRegisteredPartiesRequest_REPLACE,
					LineOfBusiness: financialservicespb.LineOfBusiness_COMMERCIAL,
				})
				return err
			},
		},
		{
			name: "ExportRegisteredParties",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.ExportRegisteredParties(ctx, &financialservicespb.ExportRegisteredPartiesRequest{
					Name:           instanceName,
					Dataset:        registeredPartiesDest,
					LineOfBusiness: financialservicespb.LineOfBusiness_COMMERCIAL,
				})
				return err
			},
		},
		{
			name: "ListDatasets",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				it := c.ListDatasets(ctx, &financialservicespb.ListDatasetsRequest{
					Parent:   instanceName,
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
			name: "GetDataset",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.GetDataset(ctx, &financialservicespb.GetDatasetRequest{Name: datasetName})
				return err
			},
		},
		{
			name: "CreateDataset",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.CreateDataset(ctx, &financialservicespb.CreateDatasetRequest{
					Parent:    instanceName,
					DatasetId: datasetID,
					Dataset:   &financialservicespb.Dataset{Name: datasetName},
				})
				return err
			},
		},
		{
			name: "UpdateDataset",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.UpdateDataset(ctx, &financialservicespb.UpdateDatasetRequest{
					Dataset: &financialservicespb.Dataset{Name: datasetName},
				})
				return err
			},
		},
		{
			name: "DeleteDataset",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.DeleteDataset(ctx, &financialservicespb.DeleteDatasetRequest{Name: datasetName})
				return err
			},
		},
		{
			name: "ListModels",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				it := c.ListModels(ctx, &financialservicespb.ListModelsRequest{
					Parent:   instanceName,
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
			name: "GetModel",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.GetModel(ctx, &financialservicespb.GetModelRequest{Name: modelName})
				return err
			},
		},
		{
			name: "CreateModel",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.CreateModel(ctx, &financialservicespb.CreateModelRequest{
					Parent:  instanceName,
					ModelId: modelID,
					Model:   &financialservicespb.Model{Name: modelName},
				})
				return err
			},
		},
		{
			name: "UpdateModel",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.UpdateModel(ctx, &financialservicespb.UpdateModelRequest{
					Model: &financialservicespb.Model{Name: modelName},
				})
				return err
			},
		},
		{
			name: "ExportModelMetadata",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.ExportModelMetadata(ctx, &financialservicespb.ExportModelMetadataRequest{
					Model:                         modelName,
					StructuredMetadataDestination: metadataDest,
				})
				return err
			},
		},
		{
			name: "DeleteModel",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.DeleteModel(ctx, &financialservicespb.DeleteModelRequest{Name: modelName})
				return err
			},
		},
		{
			name: "ListEngineConfigs",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				it := c.ListEngineConfigs(ctx, &financialservicespb.ListEngineConfigsRequest{
					Parent:   instanceName,
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
			name: "GetEngineConfig",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.GetEngineConfig(ctx, &financialservicespb.GetEngineConfigRequest{Name: engineConfigName})
				return err
			},
		},
		{
			name: "CreateEngineConfig",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.CreateEngineConfig(ctx, &financialservicespb.CreateEngineConfigRequest{
					Parent:         instanceName,
					EngineConfigId: engineConfigID,
					EngineConfig: &financialservicespb.EngineConfig{
						Name:          engineConfigName,
						EngineVersion: engineVersionName,
					},
				})
				return err
			},
		},
		{
			name: "UpdateEngineConfig",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.UpdateEngineConfig(ctx, &financialservicespb.UpdateEngineConfigRequest{
					EngineConfig: &financialservicespb.EngineConfig{Name: engineConfigName},
				})
				return err
			},
		},
		{
			name: "ExportEngineConfigMetadata",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.ExportEngineConfigMetadata(ctx, &financialservicespb.ExportEngineConfigMetadataRequest{
					EngineConfig:                  engineConfigName,
					StructuredMetadataDestination: metadataDest,
				})
				return err
			},
		},
		{
			name: "DeleteEngineConfig",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.DeleteEngineConfig(ctx, &financialservicespb.DeleteEngineConfigRequest{Name: engineConfigName})
				return err
			},
		},
		{
			name: "ListEngineVersions",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				it := c.ListEngineVersions(ctx, &financialservicespb.ListEngineVersionsRequest{
					Parent:   instanceName,
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
			name: "GetEngineVersion",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.GetEngineVersion(ctx, &financialservicespb.GetEngineVersionRequest{Name: engineVersionName})
				return err
			},
		},
		{
			name: "ListPredictionResults",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				it := c.ListPredictionResults(ctx, &financialservicespb.ListPredictionResultsRequest{
					Parent:   instanceName,
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
			name: "GetPredictionResult",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.GetPredictionResult(ctx, &financialservicespb.GetPredictionResultRequest{Name: predictionResultName})
				return err
			},
		},
		{
			name: "CreatePredictionResult",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.CreatePredictionResult(ctx, &financialservicespb.CreatePredictionResultRequest{
					Parent:             instanceName,
					PredictionResultId: predictionResultID,
					PredictionResult:   &financialservicespb.PredictionResult{Name: predictionResultName},
				})
				return err
			},
		},
		{
			name: "UpdatePredictionResult",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.UpdatePredictionResult(ctx, &financialservicespb.UpdatePredictionResultRequest{
					PredictionResult: &financialservicespb.PredictionResult{Name: predictionResultName},
				})
				return err
			},
		},
		{
			name: "ExportPredictionResultMetadata",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.ExportPredictionResultMetadata(ctx, &financialservicespb.ExportPredictionResultMetadataRequest{
					PredictionResult:              predictionResultName,
					StructuredMetadataDestination: metadataDest,
				})
				return err
			},
		},
		{
			name: "DeletePredictionResult",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.DeletePredictionResult(ctx, &financialservicespb.DeletePredictionResultRequest{Name: predictionResultName})
				return err
			},
		},
		{
			name: "ListBacktestResults",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				it := c.ListBacktestResults(ctx, &financialservicespb.ListBacktestResultsRequest{
					Parent:   instanceName,
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
			name: "GetBacktestResult",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.GetBacktestResult(ctx, &financialservicespb.GetBacktestResultRequest{Name: backtestResultName})
				return err
			},
		},
		{
			name: "CreateBacktestResult",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.CreateBacktestResult(ctx, &financialservicespb.CreateBacktestResultRequest{
					Parent:           instanceName,
					BacktestResultId: backtestResultID,
					BacktestResult:   &financialservicespb.BacktestResult{Name: backtestResultName},
				})
				return err
			},
		},
		{
			name: "UpdateBacktestResult",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.UpdateBacktestResult(ctx, &financialservicespb.UpdateBacktestResultRequest{
					BacktestResult: &financialservicespb.BacktestResult{Name: backtestResultName},
				})
				return err
			},
		},
		{
			name: "ExportBacktestResultMetadata",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.ExportBacktestResultMetadata(ctx, &financialservicespb.ExportBacktestResultMetadataRequest{
					BacktestResult:                backtestResultName,
					StructuredMetadataDestination: metadataDest,
				})
				return err
			},
		},
		{
			name: "DeleteBacktestResult",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.DeleteBacktestResult(ctx, &financialservicespb.DeleteBacktestResultRequest{Name: backtestResultName})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
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
			name: "GetOperation",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
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
			name: "CancelOperation",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *financialservices.AMLClient) error {
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
		fmt.Fprintf(os.Stderr, "warning: close financial services client: %v\n", err)
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
