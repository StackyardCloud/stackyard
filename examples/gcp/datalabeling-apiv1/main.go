package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	datalabeling "cloud.google.com/go/datalabeling/apiv1beta1"
	"cloud.google.com/go/datalabeling/apiv1beta1/datalabelingpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *datalabeling.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	datasetID := getenv("STACKYARD_GCP_DATALABELING_DATASET_ID", "team-dataset")
	annotatedDatasetID := getenv("STACKYARD_GCP_DATALABELING_ANNOTATED_DATASET_ID", "annotated-1")
	dataItemID := getenv("STACKYARD_GCP_DATALABELING_DATA_ITEM_ID", "item-1")
	exampleID := getenv("STACKYARD_GCP_DATALABELING_EXAMPLE_ID", "example-1")
	annotationSpecSetID := getenv("STACKYARD_GCP_DATALABELING_ANNOTATION_SPEC_SET_ID", "spec-set-1")
	instructionID := getenv("STACKYARD_GCP_DATALABELING_INSTRUCTION_ID", "instruction-1")
	evaluationID := getenv("STACKYARD_GCP_DATALABELING_EVALUATION_ID", "evaluation-1")
	evaluationJobID := getenv("STACKYARD_GCP_DATALABELING_EVALUATION_JOB_ID", "job-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	datasetName := projectName + "/datasets/" + datasetID
	annotatedDatasetName := datasetName + "/annotatedDatasets/" + annotatedDatasetID
	dataItemName := datasetName + "/dataItems/" + dataItemID
	exampleName := annotatedDatasetName + "/examples/" + exampleID
	annotationSpecSetName := projectName + "/annotationSpecSets/" + annotationSpecSetID
	instructionName := projectName + "/instructions/" + instructionID
	evaluationName := datasetName + "/evaluations/" + evaluationID
	evaluationJobName := projectName + "/evaluationJobs/" + evaluationJobID

	fmt.Printf("Stackyard GCP Data Labeling apiv1beta1 client using %s\n", apiEndpoint)

	client, err := datalabeling.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create datalabeling client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListDatasets",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				it := c.ListDatasets(ctx, &datalabelingpb.ListDatasetsRequest{
					Parent:   projectName,
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
			name: "CreateDataset",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.CreateDataset(ctx, &datalabelingpb.CreateDatasetRequest{
					Parent: projectName,
					Dataset: &datalabelingpb.Dataset{
						DisplayName: "Team Dataset",
						Description: "Stackyard Data Labeling dataset",
					},
				})
				return err
			},
		},
		{
			name: "GetDataset",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.GetDataset(ctx, &datalabelingpb.GetDatasetRequest{Name: datasetName})
				return err
			},
		},
		{
			name: "ImportData",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.ImportData(ctx, &datalabelingpb.ImportDataRequest{
					Name: datasetName,
					InputConfig: &datalabelingpb.InputConfig{
						DataType: datalabelingpb.DataType_IMAGE,
						Source: &datalabelingpb.InputConfig_GcsSource{
							GcsSource: &datalabelingpb.GcsSource{
								InputUri: "gs://stackyard-datalabeling/input.csv",
								MimeType: "text/csv",
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "ExportData",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.ExportData(ctx, &datalabelingpb.ExportDataRequest{
					Name:             datasetName,
					AnnotatedDataset: annotatedDatasetName,
					OutputConfig: &datalabelingpb.OutputConfig{
						Destination: &datalabelingpb.OutputConfig_GcsDestination{
							GcsDestination: &datalabelingpb.GcsDestination{
								OutputUri: "gs://stackyard-datalabeling/export.jsonl",
								MimeType:  "application/json",
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "ListDataItems",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				it := c.ListDataItems(ctx, &datalabelingpb.ListDataItemsRequest{
					Parent:   datasetName,
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
			name: "GetDataItem",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.GetDataItem(ctx, &datalabelingpb.GetDataItemRequest{Name: dataItemName})
				return err
			},
		},
		{
			name: "ListAnnotatedDatasets",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				it := c.ListAnnotatedDatasets(ctx, &datalabelingpb.ListAnnotatedDatasetsRequest{
					Parent:   datasetName,
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
			name: "GetAnnotatedDataset",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.GetAnnotatedDataset(ctx, &datalabelingpb.GetAnnotatedDatasetRequest{Name: annotatedDatasetName})
				return err
			},
		},
		{
			name: "LabelImage",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.LabelImage(ctx, &datalabelingpb.LabelImageRequest{
					Parent: datasetName,
					BasicConfig: &datalabelingpb.HumanAnnotationConfig{
						Instruction:                 instructionName,
						AnnotatedDatasetDisplayName: "image-label-run",
					},
					Feature: datalabelingpb.LabelImageRequest_CLASSIFICATION,
					RequestConfig: &datalabelingpb.LabelImageRequest_ImageClassificationConfig{
						ImageClassificationConfig: &datalabelingpb.ImageClassificationConfig{
							AnnotationSpecSet: annotationSpecSetName,
						},
					},
				})
				return err
			},
		},
		{
			name: "LabelVideo",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.LabelVideo(ctx, &datalabelingpb.LabelVideoRequest{
					Parent: datasetName,
					BasicConfig: &datalabelingpb.HumanAnnotationConfig{
						Instruction:                 instructionName,
						AnnotatedDatasetDisplayName: "video-label-run",
					},
					Feature: datalabelingpb.LabelVideoRequest_CLASSIFICATION,
					RequestConfig: &datalabelingpb.LabelVideoRequest_VideoClassificationConfig{
						VideoClassificationConfig: &datalabelingpb.VideoClassificationConfig{
							AnnotationSpecSetConfigs: []*datalabelingpb.VideoClassificationConfig_AnnotationSpecSetConfig{
								{AnnotationSpecSet: annotationSpecSetName},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "LabelText",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.LabelText(ctx, &datalabelingpb.LabelTextRequest{
					Parent: datasetName,
					BasicConfig: &datalabelingpb.HumanAnnotationConfig{
						Instruction:                 instructionName,
						AnnotatedDatasetDisplayName: "text-label-run",
					},
					Feature: datalabelingpb.LabelTextRequest_TEXT_CLASSIFICATION,
					RequestConfig: &datalabelingpb.LabelTextRequest_TextClassificationConfig{
						TextClassificationConfig: &datalabelingpb.TextClassificationConfig{
							AnnotationSpecSet: annotationSpecSetName,
						},
					},
				})
				return err
			},
		},
		{
			name: "ListExamples",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				it := c.ListExamples(ctx, &datalabelingpb.ListExamplesRequest{
					Parent:   annotatedDatasetName,
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
			name: "GetExample",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.GetExample(ctx, &datalabelingpb.GetExampleRequest{Name: exampleName})
				return err
			},
		},
		{
			name: "CreateAnnotationSpecSet",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.CreateAnnotationSpecSet(ctx, &datalabelingpb.CreateAnnotationSpecSetRequest{
					Parent: projectName,
					AnnotationSpecSet: &datalabelingpb.AnnotationSpecSet{
						DisplayName: "Team Labels",
						AnnotationSpecs: []*datalabelingpb.AnnotationSpec{
							{DisplayName: "cat"},
							{DisplayName: "dog"},
						},
					},
				})
				return err
			},
		},
		{
			name: "ListAnnotationSpecSets",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				it := c.ListAnnotationSpecSets(ctx, &datalabelingpb.ListAnnotationSpecSetsRequest{
					Parent:   projectName,
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
			name: "GetAnnotationSpecSet",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.GetAnnotationSpecSet(ctx, &datalabelingpb.GetAnnotationSpecSetRequest{Name: annotationSpecSetName})
				return err
			},
		},
		{
			name: "CreateInstruction",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.CreateInstruction(ctx, &datalabelingpb.CreateInstructionRequest{
					Parent: projectName,
					Instruction: &datalabelingpb.Instruction{
						DisplayName: "Team Labeling Instruction",
						DataType:    datalabelingpb.DataType_IMAGE,
						PdfInstruction: &datalabelingpb.PdfInstruction{
							GcsFileUri: "gs://stackyard-datalabeling/instructions/guide.pdf",
						},
					},
				})
				return err
			},
		},
		{
			name: "ListInstructions",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				it := c.ListInstructions(ctx, &datalabelingpb.ListInstructionsRequest{
					Parent:   projectName,
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
			name: "GetInstruction",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.GetInstruction(ctx, &datalabelingpb.GetInstructionRequest{Name: instructionName})
				return err
			},
		},
		{
			name: "GetEvaluation",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.GetEvaluation(ctx, &datalabelingpb.GetEvaluationRequest{Name: evaluationName})
				return err
			},
		},
		{
			name: "SearchEvaluations",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				it := c.SearchEvaluations(ctx, &datalabelingpb.SearchEvaluationsRequest{
					Parent:   projectName,
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
			name: "SearchExampleComparisons",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				it := c.SearchExampleComparisons(ctx, &datalabelingpb.SearchExampleComparisonsRequest{
					Parent:   evaluationName,
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
			name: "CreateEvaluationJob",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.CreateEvaluationJob(ctx, &datalabelingpb.CreateEvaluationJobRequest{
					Parent: projectName,
					Job: &datalabelingpb.EvaluationJob{
						Description:             "Stackyard evaluation job",
						Schedule:                "every 24 hours",
						ModelVersion:            fmt.Sprintf("projects/%s/models/image-model/versions/v1", projectID),
						AnnotationSpecSet:       annotationSpecSetName,
						LabelMissingGroundTruth: true,
						EvaluationJobConfig: &datalabelingpb.EvaluationJobConfig{
							InputConfig: &datalabelingpb.InputConfig{
								DataType:       datalabelingpb.DataType_IMAGE,
								AnnotationType: datalabelingpb.AnnotationType_IMAGE_CLASSIFICATION_ANNOTATION,
								ClassificationMetadata: &datalabelingpb.ClassificationMetadata{
									IsMultiLabel: false,
								},
								Source: &datalabelingpb.InputConfig_BigquerySource{
									BigquerySource: &datalabelingpb.BigQuerySource{
										InputUri: fmt.Sprintf("bq://%s/datalabeling/predictions", projectID),
									},
								},
							},
							EvaluationConfig: &datalabelingpb.EvaluationConfig{},
							HumanAnnotationConfig: &datalabelingpb.HumanAnnotationConfig{
								Instruction:                 instructionName,
								AnnotatedDatasetDisplayName: "eval-job-annotations",
							},
							HumanAnnotationRequestConfig: &datalabelingpb.EvaluationJobConfig_ImageClassificationConfig{
								ImageClassificationConfig: &datalabelingpb.ImageClassificationConfig{
									AnnotationSpecSet: annotationSpecSetName,
								},
							},
							BigqueryImportKeys: map[string]string{
								"data_json_key":        "data",
								"label_json_key":       "predicted_label",
								"label_score_json_key": "score",
							},
							ExampleCount:            10,
							ExampleSamplePercentage: 0.1,
						},
					},
				})
				return err
			},
		},
		{
			name: "ListEvaluationJobs",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				it := c.ListEvaluationJobs(ctx, &datalabelingpb.ListEvaluationJobsRequest{
					Parent:   projectName,
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
			name: "GetEvaluationJob",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.GetEvaluationJob(ctx, &datalabelingpb.GetEvaluationJobRequest{Name: evaluationJobName})
				return err
			},
		},
		{
			name: "PauseEvaluationJob",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				return c.PauseEvaluationJob(ctx, &datalabelingpb.PauseEvaluationJobRequest{Name: evaluationJobName})
			},
		},
		{
			name: "ResumeEvaluationJob",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				return c.ResumeEvaluationJob(ctx, &datalabelingpb.ResumeEvaluationJobRequest{Name: evaluationJobName})
			},
		},
		{
			name: "UpdateEvaluationJob",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				_, err := c.UpdateEvaluationJob(ctx, &datalabelingpb.UpdateEvaluationJobRequest{
					EvaluationJob: &datalabelingpb.EvaluationJob{
						Name: evaluationJobName,
						EvaluationJobConfig: &datalabelingpb.EvaluationJobConfig{
							HumanAnnotationConfig: &datalabelingpb.HumanAnnotationConfig{
								Instruction:                 instructionName,
								AnnotatedDatasetDisplayName: "eval-job-annotations-updated",
							},
							ExampleCount: 25,
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{
							"evaluation_job_config.example_count",
							"evaluation_job_config.human_annotation_config.instruction",
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteEvaluationJob",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				return c.DeleteEvaluationJob(ctx, &datalabelingpb.DeleteEvaluationJobRequest{Name: evaluationJobName})
			},
		},
		{
			name: "DeleteAnnotatedDataset",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				return c.DeleteAnnotatedDataset(ctx, &datalabelingpb.DeleteAnnotatedDatasetRequest{Name: annotatedDatasetName})
			},
		},
		{
			name: "DeleteAnnotationSpecSet",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				return c.DeleteAnnotationSpecSet(ctx, &datalabelingpb.DeleteAnnotationSpecSetRequest{Name: annotationSpecSetName})
			},
		},
		{
			name: "DeleteInstruction",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				return c.DeleteInstruction(ctx, &datalabelingpb.DeleteInstructionRequest{Name: instructionName})
			},
		},
		{
			name: "DeleteDataset",
			call: func(ctx context.Context, c *datalabeling.Client) error {
				return c.DeleteDataset(ctx, &datalabelingpb.DeleteDatasetRequest{Name: datasetName})
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
		fmt.Fprintf(os.Stderr, "warning: close datalabeling client: %v\n", err)
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
