package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	automl "cloud.google.com/go/automl/apiv1"
	"cloud.google.com/go/automl/apiv1/automlpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *automl.Client, *automl.PredictionClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	location := getenv("STACKYARD_GCP_LOCATION", "projects/stackyard/locations/us-central1")
	datasetID := getenv("STACKYARD_GCP_DATASET_ID", "team-dataset")
	modelID := getenv("STACKYARD_GCP_MODEL_ID", "team-model")
	modelEvalID := getenv("STACKYARD_GCP_MODEL_EVALUATION_ID", "eval-1")

	datasetName := location + "/datasets/" + datasetID
	modelName := location + "/models/" + modelID
	modelEvaluationName := modelName + "/modelEvaluations/" + modelEvalID

	fmt.Printf("Stackyard GCP AutoML apiv1 clients using %s\n", apiEndpoint)

	autoMLClient, err := automl.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create automl client: %v", err)
	}
	defer closeClient("automl", autoMLClient.Close)

	predictionClient, err := automl.NewPredictionRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create automl prediction client: %v", err)
	}
	defer closeClient("automl prediction", predictionClient.Close)

	calls := []callSpec{
		{
			name: "ListDatasets",
			call: func(ctx context.Context, c *automl.Client, _ *automl.PredictionClient) error {
				it := c.ListDatasets(ctx, &automlpb.ListDatasetsRequest{
					Parent:   location,
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
			call: func(ctx context.Context, c *automl.Client, _ *automl.PredictionClient) error {
				_, err := c.GetDataset(ctx, &automlpb.GetDatasetRequest{Name: datasetName})
				return err
			},
		},
		{
			name: "ListModels",
			call: func(ctx context.Context, c *automl.Client, _ *automl.PredictionClient) error {
				it := c.ListModels(ctx, &automlpb.ListModelsRequest{
					Parent:   location,
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
			call: func(ctx context.Context, c *automl.Client, _ *automl.PredictionClient) error {
				_, err := c.GetModel(ctx, &automlpb.GetModelRequest{Name: modelName})
				return err
			},
		},
		{
			name: "ListModelEvaluations",
			call: func(ctx context.Context, c *automl.Client, _ *automl.PredictionClient) error {
				it := c.ListModelEvaluations(ctx, &automlpb.ListModelEvaluationsRequest{
					Parent:   modelName,
					Filter:   "annotation_spec_id:*",
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
			name: "GetModelEvaluation",
			call: func(ctx context.Context, c *automl.Client, _ *automl.PredictionClient) error {
				_, err := c.GetModelEvaluation(ctx, &automlpb.GetModelEvaluationRequest{Name: modelEvaluationName})
				return err
			},
		},
		{
			name: "Predict",
			call: func(ctx context.Context, _ *automl.Client, p *automl.PredictionClient) error {
				_, err := p.Predict(ctx, &automlpb.PredictRequest{
					Name: modelName,
					Payload: &automlpb.ExamplePayload{
						Payload: &automlpb.ExamplePayload_TextSnippet{
							TextSnippet: &automlpb.TextSnippet{
								Content:  "hello from stackyard",
								MimeType: "text/plain",
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "BatchPredict",
			call: func(ctx context.Context, _ *automl.Client, p *automl.PredictionClient) error {
				_, err := p.BatchPredict(ctx, &automlpb.BatchPredictRequest{
					Name: modelName,
					InputConfig: &automlpb.BatchPredictInputConfig{
						Source: &automlpb.BatchPredictInputConfig_GcsSource{
							GcsSource: &automlpb.GcsSource{
								InputUris: []string{"gs://stackyard-input/predict.csv"},
							},
						},
					},
					OutputConfig: &automlpb.BatchPredictOutputConfig{
						Destination: &automlpb.BatchPredictOutputConfig_GcsDestination{
							GcsDestination: &automlpb.GcsDestination{
								OutputUriPrefix: "gs://stackyard-output/predictions",
							},
						},
					},
				})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, autoMLClient, predictionClient)
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
