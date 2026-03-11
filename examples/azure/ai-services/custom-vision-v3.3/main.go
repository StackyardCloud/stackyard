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

const customVisionAPIVersion = "3.3"

type customVisionClient struct {
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

func newCustomVisionClient(endpoint, account, subscriptionKey string) *customVisionClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"custom-vision-v3.3",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &customVisionClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *customVisionClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	projectID := getenv("STACKYARD_AZURE_CUSTOM_VISION_PROJECT_ID", "project-a")
	iterationID := getenv("STACKYARD_AZURE_CUSTOM_VISION_ITERATION_ID", "iteration-a")
	predictionID := getenv("STACKYARD_AZURE_CUSTOM_VISION_PREDICTION_ID", "prediction-a")
	tagID := getenv("STACKYARD_AZURE_CUSTOM_VISION_TAG_ID", "tag-a")

	fmt.Printf("Stackyard Azure Custom Vision (custom-vision-v3.3) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCustomVisionClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListDomains",
			method:   http.MethodGet,
			path:     "/azure/customvision/v3.3/training/domains?api-version=" + customVisionAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateProject",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects?api-version=" + customVisionAPIVersion,
			payload: map[string]any{
				"name":               "stackyard-project",
				"classificationType": "Multiclass",
				"domainId":           "b30a91ae-e3c1-4f73-a81e-c270bff27c39",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetProject",
			method:   http.MethodGet,
			path:     "/azure/customvision/v3.3/training/projects/" + projectID + "?api-version=" + customVisionAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "TrainProject",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/" + projectID + "/train?api-version=" + customVisionAPIVersion,
			payload: map[string]any{
				"trainingType": "Regular",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateImagesFromFiles",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/" + projectID + "/images/files?api-version=" + customVisionAPIVersion,
			payload: map[string]any{
				"images": []map[string]any{
					{
						"name":     "image-a.jpg",
						"contents": "aGVsbG8=",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "QueryImages",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/" + projectID + "/images/query?api-version=" + customVisionAPIVersion,
			payload: map[string]any{
				"take":    16,
				"orderBy": "Newest",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "PublishIteration",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/" + projectID + "/iterations/" + iterationID + "/publish?api-version=" + customVisionAPIVersion,
			payload: map[string]any{
				"publishName":  "stackyard-publish",
				"predictionId": "stackyard-resource-id",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "QueryPredictionResults",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/" + projectID + "/predictions/query?api-version=" + customVisionAPIVersion,
			payload: map[string]any{
				"maxCount": 10,
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "DeletePrediction",
			method:   http.MethodDelete,
			path:     "/azure/customvision/v3.3/training/projects/" + projectID + "/predictions/" + predictionID + "?api-version=" + customVisionAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateTag",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/" + projectID + "/tags?api-version=" + customVisionAPIVersion,
			payload: map[string]any{
				"name": "stackyard-tag",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "UpdateTag",
			method: http.MethodPatch,
			path:   "/azure/customvision/v3.3/training/projects/" + projectID + "/tags/" + tagID + "?api-version=" + customVisionAPIVersion,
			payload: map[string]any{
				"name": "stackyard-tag-updated",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "QuickTestImage",
			method: http.MethodPost,
			path:   "/azure/customvision/v3.3/training/projects/" + projectID + "/quicktest/image?api-version=" + customVisionAPIVersion,
			payload: map[string]any{
				"imageData": "aGVsbG8=",
			},
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
		fmt.Println("All custom-vision routes are staged in this Stackyard build.")
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
