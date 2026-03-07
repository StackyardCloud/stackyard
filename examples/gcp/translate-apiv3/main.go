package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	translate "cloud.google.com/go/translate/apiv3"
	translatepb "cloud.google.com/go/translate/apiv3/translatepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	ctx := context.Background()

	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	apiEndpoint := endpoint + "/gcp"
	grpcEndpoint := strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	glossaryID := getenv("STACKYARD_GCP_TRANSLATE_GLOSSARY_ID", "glossary-1")
	datasetID := getenv("STACKYARD_GCP_TRANSLATE_DATASET_ID", "dataset-1")
	adaptiveDatasetID := getenv("STACKYARD_GCP_TRANSLATE_ADAPTIVE_DATASET_ID", "adaptive-dataset-1")
	modelID := getenv("STACKYARD_GCP_TRANSLATE_MODEL_ID", "model-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	projectName := fmt.Sprintf("projects/%s", projectID)
	glossaryName := fmt.Sprintf("%s/glossaries/%s", parent, glossaryID)
	datasetName := fmt.Sprintf("%s/datasets/%s", parent, datasetID)
	modelName := fmt.Sprintf("%s/models/%s", parent, modelID)
	adaptiveDatasetName := fmt.Sprintf("%s/adaptiveMtDatasets/%s", parent, adaptiveDatasetID)

	fmt.Printf("Stackyard GCP Translation V3 translate/apiv3 client using %s (grpc=%s)\n", apiEndpoint, grpcEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, projectID, locationID); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	client, err := translate.NewTranslationClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create translate v3 client: %v", err)
	}
	defer closeClient(client.Close)

	locationIt := client.ListLocations(ctx, &locationpb.ListLocationsRequest{
		Name:     projectName,
		PageSize: 1,
	})
	if _, err := locationIt.Next(); err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListLocations failed: %v", err)
	}
	logf("ListLocations succeeded")

	if _, err := client.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent}); err != nil {
		exitf("GetLocation failed: %v", err)
	}
	logf("GetLocation succeeded")

	translateResp, err := client.TranslateText(ctx, &translatepb.TranslateTextRequest{
		Parent:             parent,
		Contents:           []string{"hello stackyard"},
		TargetLanguageCode: "es",
	})
	if err != nil {
		exitf("TranslateText failed: %v", err)
	}
	if len(translateResp.GetTranslations()) == 0 {
		exitf("TranslateText returned empty translations")
	}
	logf("TranslateText succeeded")

	detectResp, err := client.DetectLanguage(ctx, &translatepb.DetectLanguageRequest{
		Parent: parent,
		Source: &translatepb.DetectLanguageRequest_Content{
			Content: "detect me",
		},
	})
	if err != nil {
		exitf("DetectLanguage failed: %v", err)
	}
	if len(detectResp.GetLanguages()) == 0 {
		exitf("DetectLanguage returned empty languages")
	}
	logf("DetectLanguage succeeded")

	supportedResp, err := client.GetSupportedLanguages(ctx, &translatepb.GetSupportedLanguagesRequest{
		Parent:              parent,
		DisplayLanguageCode: "en",
	})
	if err != nil {
		exitf("GetSupportedLanguages failed: %v", err)
	}
	if len(supportedResp.GetLanguages()) == 0 {
		exitf("GetSupportedLanguages returned empty languages")
	}
	logf("GetSupportedLanguages succeeded")

	createGlossaryOp, err := client.CreateGlossary(ctx, &translatepb.CreateGlossaryRequest{
		Parent: parent,
		Glossary: &translatepb.Glossary{
			Name:        glossaryName,
			DisplayName: "Stackyard Glossary",
		},
	})
	if err != nil {
		exitf("CreateGlossary failed: %v", err)
	}
	if strings.TrimSpace(createGlossaryOp.Name()) == "" {
		exitf("CreateGlossary returned empty operation name")
	}
	logf("CreateGlossary succeeded")

	glossaryIt := client.ListGlossaries(ctx, &translatepb.ListGlossariesRequest{
		Parent:   parent,
		PageSize: 1,
	})
	if _, err := glossaryIt.Next(); err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListGlossaries failed: %v", err)
	}
	logf("ListGlossaries succeeded")

	createDatasetOp, err := client.CreateDataset(ctx, &translatepb.CreateDatasetRequest{
		Parent: parent,
		Dataset: &translatepb.Dataset{
			Name:               datasetName,
			DisplayName:        "Stackyard Dataset",
			SourceLanguageCode: "en",
			TargetLanguageCode: "es",
		},
	})
	if err != nil {
		exitf("CreateDataset failed: %v", err)
	}
	if strings.TrimSpace(createDatasetOp.Name()) == "" {
		exitf("CreateDataset returned empty operation name")
	}
	logf("CreateDataset succeeded")

	if _, err := client.CreateAdaptiveMtDataset(ctx, &translatepb.CreateAdaptiveMtDatasetRequest{
		Parent: parent,
		AdaptiveMtDataset: &translatepb.AdaptiveMtDataset{
			Name:               adaptiveDatasetName,
			DisplayName:        "Adaptive Dataset",
			SourceLanguageCode: "en",
			TargetLanguageCode: "es",
		},
	}); err != nil {
		exitf("CreateAdaptiveMtDataset failed: %v", err)
	}
	logf("CreateAdaptiveMtDataset succeeded")

	adaptiveIt := client.ListAdaptiveMtDatasets(ctx, &translatepb.ListAdaptiveMtDatasetsRequest{
		Parent:   parent,
		PageSize: 1,
	})
	if _, err := adaptiveIt.Next(); err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListAdaptiveMtDatasets failed: %v", err)
	}
	logf("ListAdaptiveMtDatasets succeeded")

	createModelOp, err := client.CreateModel(ctx, &translatepb.CreateModelRequest{
		Parent: parent,
		Model: &translatepb.Model{
			Name:        modelName,
			DisplayName: "Stackyard Model",
			Dataset:     datasetName,
		},
	})
	if err != nil {
		exitf("CreateModel failed: %v", err)
	}
	if strings.TrimSpace(createModelOp.Name()) == "" {
		exitf("CreateModel returned empty operation name")
	}
	logf("CreateModel succeeded")

	modelIt := client.ListModels(ctx, &translatepb.ListModelsRequest{
		Parent:   parent,
		PageSize: 1,
	})
	if _, err := modelIt.Next(); err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListModels failed: %v", err)
	}
	logf("ListModels succeeded")

	batchOp, err := client.BatchTranslateText(ctx, &translatepb.BatchTranslateTextRequest{
		Parent:             parent,
		SourceLanguageCode: "en",
		TargetLanguageCodes: []string{
			"es",
		},
		InputConfigs: []*translatepb.InputConfig{
			{
				MimeType: "text/plain",
			},
		},
		OutputConfig: &translatepb.OutputConfig{},
	})
	if err != nil {
		exitf("BatchTranslateText failed: %v", err)
	}
	if strings.TrimSpace(batchOp.Name()) == "" {
		exitf("BatchTranslateText returned empty operation name")
	}
	logf("BatchTranslateText succeeded")

	_, err = client.TranslateText(ctx, &translatepb.TranslateTextRequest{
		Parent:   parent,
		Contents: []string{"missing target language"},
	})
	if err == nil {
		exitf("TranslateText validation call unexpectedly succeeded")
	}
	if !isExpectedInvalidArgument(err) {
		exitf("TranslateText validation call returned unexpected error: %v", err)
	}
	logf("TranslateText validation call returned InvalidArgument (expected)")

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, apiEndpoint, projectID, locationID string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	probeURL := fmt.Sprintf("%s/v3/projects/%s/locations/%s/translate?stackyard_contract_probe=1&typedSuccess=1", apiEndpoint, projectID, locationID)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "translate")

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("translate contract probe did not become ready: %s", probeURL)
}

func isExpectedInvalidArgument(err error) bool {
	if err == nil {
		return false
	}
	if grpcStatus, ok := status.FromError(err); ok {
		return grpcStatus.Code() == codes.InvalidArgument
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusBadRequest && strings.Contains(strings.ToLower(apiErr.Message), "invalid")
	}
	return false
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close translate v3 client: %v\n", err)
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
