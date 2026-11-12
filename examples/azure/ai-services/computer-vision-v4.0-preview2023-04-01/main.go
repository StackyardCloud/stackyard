package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const computerVisionAPIVersion = "2023-04-01-preview"

type computerVisionClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type sharedKeyAndSubscriptionPolicy struct {
	account         string
	subscriptionKey string
}

func (p *sharedKeyAndSubscriptionPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newComputerVisionClient(endpoint, account, subscriptionKey string) *computerVisionClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"computer-vision-v4.0-preview2023-04-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &computerVisionClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *computerVisionClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

	if payload != nil {
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body %s %s: %w", method, path, readErr)
	}

	if len(expectedStatuses) == 0 {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else {
		matched := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				matched = true
				break
			}
		}
		if !matched {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	}

	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	datasetName := getenv("STACKYARD_AZURE_COMPUTER_VISION_DATASET", "dataset-a")
	modelName := getenv("STACKYARD_AZURE_COMPUTER_VISION_MODEL", "model-a")
	evaluationName := getenv("STACKYARD_AZURE_COMPUTER_VISION_EVALUATION", "eval-a")
	runName := getenv("STACKYARD_AZURE_COMPUTER_VISION_RUN", "run-a")

	fmt.Printf("Stackyard Azure Computer Vision (computer-vision-v4.0-preview2023-04-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newComputerVisionClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "CreateDataset",
			method: http.MethodPut,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/datasets/" + datasetName + "?api-version=" + computerVisionAPIVersion,
			payload: map[string]any{
				"displayName": datasetName,
				"description": "stackyard dataset",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListDatasets",
			method:   http.MethodGet,
			path:     "/azure/computervision/v4.0-preview/2023-04-01/datasets?api-version=" + computerVisionAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "AnalyzeImage",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imageanalysis:analyze?api-version=" + computerVisionAPIVersion,
			payload: map[string]any{
				"url":      "https://example.com/image.jpg",
				"features": []string{"caption"},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "StitchImages",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imagecomposition:stitch?api-version=" + computerVisionAPIVersion,
			payload: map[string]any{
				"images": []string{"https://example.com/1.jpg", "https://example.com/2.jpg"},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "VectorizeText",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imageretrieval:vectorizetext?api-version=" + computerVisionAPIVersion,
			payload: map[string]any{
				"text": "retail shelf compliance",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateModel",
			method: http.MethodPut,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/models/" + modelName + "?api-version=" + computerVisionAPIVersion,
			payload: map[string]any{
				"kind":        "productRecognition",
				"description": "stackyard model",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateModelEvaluation",
			method: http.MethodPut,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/modelevaluations/" + evaluationName + "?api-version=" + computerVisionAPIVersion,
			payload: map[string]any{
				"dataset": datasetName,
				"model":   modelName,
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "MatchPlanogram",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/planogramcompliance:match?api-version=" + computerVisionAPIVersion,
			payload: map[string]any{
				"planogram": map[string]any{"items": []any{}},
				"observed":  map[string]any{"items": []any{}},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateProductRecognitionRun",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/productrecognition/runs?api-version=" + computerVisionAPIVersion,
			payload: map[string]any{
				"modelName": modelName,
				"image":     "https://example.com/shelf.jpg",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetProductRecognitionRun",
			method:   http.MethodGet,
			path:     "/azure/computervision/v4.0-preview/2023-04-01/productrecognition/runs/" + runName + "?api-version=" + computerVisionAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		_, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		if status == http.StatusNotImplemented {
			notImplementedCount++
			fmt.Printf("Route is recognized but not implemented yet: %s\n", call.path)
			continue
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All computer vision routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
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
