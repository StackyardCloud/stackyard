package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dataqna "cloud.google.com/go/dataqna/apiv1alpha"
	"cloud.google.com/go/dataqna/apiv1alpha/dataqnapb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	questionID := getenv("STACKYARD_GCP_DATAQNA_QUESTION_ID", "question-1")
	datasetID := getenv("STACKYARD_GCP_DATAQNA_DATASET_ID", "analytics")
	tableID := getenv("STACKYARD_GCP_DATAQNA_TABLE_ID", "orders")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	questionName := parent + "/questions/" + questionID
	feedbackName := questionName + "/userFeedback"
	scope := fmt.Sprintf("//bigquery.googleapis.com/projects/%s/datasets/%s/tables/%s", projectID, datasetID, tableID)

	fmt.Printf("Stackyard GCP Data QnA apiv1alpha clients using %s\n", apiEndpoint)

	autoSuggestionClient, err := dataqna.NewAutoSuggestionRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create auto suggestion client: %v", err)
	}
	defer closeClient("auto suggestion", autoSuggestionClient.Close)

	questionClient, err := dataqna.NewQuestionRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create question client: %v", err)
	}
	defer closeClient("question", questionClient.Close)

	calls := []callSpec{
		{
			name: "SuggestQueries",
			call: func(ctx context.Context) error {
				_, err := autoSuggestionClient.SuggestQueries(ctx, &dataqnapb.SuggestQueriesRequest{
					Parent: parent,
					Scopes: []string{scope},
					Query:  "top orders by total amount",
					SuggestionTypes: []dataqnapb.SuggestionType{
						dataqnapb.SuggestionType_ENTITY,
						dataqnapb.SuggestionType_TEMPLATE,
					},
				})
				return err
			},
		},
		{
			name: "CreateQuestion",
			call: func(ctx context.Context) error {
				_, err := questionClient.CreateQuestion(ctx, &dataqnapb.CreateQuestionRequest{
					Parent: parent,
					Question: &dataqnapb.Question{
						Scopes: []string{scope},
						Query:  "total amount by region",
					},
				})
				return err
			},
		},
		{
			name: "GetQuestion",
			call: func(ctx context.Context) error {
				_, err := questionClient.GetQuestion(ctx, &dataqnapb.GetQuestionRequest{
					Name: questionName,
				})
				return err
			},
		},
		{
			name: "ExecuteQuestion",
			call: func(ctx context.Context) error {
				_, err := questionClient.ExecuteQuestion(ctx, &dataqnapb.ExecuteQuestionRequest{
					Name:                questionName,
					InterpretationIndex: 0,
				})
				return err
			},
		},
		{
			name: "GetUserFeedback",
			call: func(ctx context.Context) error {
				_, err := questionClient.GetUserFeedback(ctx, &dataqnapb.GetUserFeedbackRequest{
					Name: feedbackName,
				})
				return err
			},
		},
		{
			name: "UpdateUserFeedback",
			call: func(ctx context.Context) error {
				_, err := questionClient.UpdateUserFeedback(ctx, &dataqnapb.UpdateUserFeedbackRequest{
					UserFeedback: &dataqnapb.UserFeedback{
						Name:             feedbackName,
						FreeFormFeedback: "Query interpretation matched the expected metric.",
						Rating:           dataqnapb.UserFeedback_POSITIVE,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"free_form_feedback", "rating"},
					},
				})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx)
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
