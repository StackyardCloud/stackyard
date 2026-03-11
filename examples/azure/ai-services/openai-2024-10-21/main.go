package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const openAIAPIVersion = "2024-10-21"

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	fileID := getenv("STACKYARD_AZURE_OPENAI_FILE_ID", "file-abc")
	batchID := getenv("STACKYARD_AZURE_OPENAI_BATCH_ID", "batch-123")
	fineTuningJobID := getenv("STACKYARD_AZURE_OPENAI_FINE_TUNING_JOB_ID", "ftjob-123")
	uploadID := getenv("STACKYARD_AZURE_OPENAI_UPLOAD_ID", "upload-123")
	modelName := getenv("STACKYARD_AZURE_OPENAI_MODEL_NAME", "gpt-4o-mini")

	fmt.Printf("Stackyard Azure OpenAI typed SDK example using %s\n", endpoint)

	requestOptions := []option.RequestOption{
		option.WithBaseURL(endpoint + "/azure/openai"),
		option.WithHeader("Authorization", "SharedKey "+account+":signature"),
		option.WithQuery("api-version", openAIAPIVersion),
	}
	if strings.TrimSpace(subscriptionKey) != "" {
		requestOptions = append(requestOptions, option.WithHeader("Ocp-Apim-Subscription-Key", subscriptionKey))
	}
	client := openai.NewClient(requestOptions...)

	calls := []struct {
		name string
		run  func() error
	}{
		{
			name: "UploadFile",
			run: func() error {
				_, err := client.Files.New(ctx, openai.FileNewParams{
					File:    strings.NewReader("{\"messages\":[]}"),
					Purpose: openai.FilePurposeAssistants,
				})
				return err
			},
		},
		{
			name: "GetFile",
			run: func() error {
				_, err := client.Files.Get(ctx, fileID)
				return err
			},
		},
		{
			name: "CreateBatch",
			run: func() error {
				_, err := client.Batches.New(ctx, openai.BatchNewParams{
					CompletionWindow: openai.BatchNewParamsCompletionWindow24h,
					Endpoint:         openai.BatchNewParamsEndpointV1ChatCompletions,
					InputFileID:      fileID,
				})
				return err
			},
		},
		{
			name: "GetBatch",
			run: func() error {
				_, err := client.Batches.Get(ctx, batchID)
				return err
			},
		},
		{
			name: "CancelBatch",
			run: func() error {
				_, err := client.Batches.Cancel(ctx, batchID)
				return err
			},
		},
		{
			name: "CreateFineTuningJob",
			run: func() error {
				_, err := client.FineTuning.Jobs.New(ctx, openai.FineTuningJobNewParams{
					Model:        openai.FineTuningJobNewParamsModel(modelName),
					TrainingFile: fileID,
				})
				return err
			},
		},
		{
			name: "ListFineTuningEvents",
			run: func() error {
				_, err := client.FineTuning.Jobs.ListEvents(ctx, fineTuningJobID, openai.FineTuningJobListEventsParams{})
				return err
			},
		},
		{
			name: "ListModels",
			run: func() error {
				_, err := client.Models.List(ctx)
				return err
			},
		},
		{
			name: "GetModel",
			run: func() error {
				_, err := client.Models.Get(ctx, modelName)
				return err
			},
		},
		{
			name: "AddUploadPart",
			run: func() error {
				_, err := client.Uploads.Parts.New(ctx, uploadID, openai.UploadPartNewParams{
					Data: strings.NewReader("part-data"),
				})
				return err
			},
		},
		{
			name: "CompleteUpload",
			run: func() error {
				_, err := client.Uploads.Complete(ctx, uploadID, openai.UploadCompleteParams{
					PartIDs: []string{"part-1", "part-2"},
				})
				return err
			},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		err := call.run()
		switch {
		case err == nil:
			fmt.Printf("%s: ok\n", call.name)
		case isNotImplemented(err):
			notImplementedCount++
			fmt.Printf("Route is recognized but not implemented yet: %s\n", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All openai routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
}

func isNotImplemented(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == http.StatusNotImplemented {
			return true
		}
		if strings.EqualFold(strings.TrimSpace(apiErr.Code), "NotImplemented") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
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
