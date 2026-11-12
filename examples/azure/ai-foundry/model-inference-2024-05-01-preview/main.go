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

const modelInferenceAPIVersion = "2024-05-01-preview"

type modelInferenceClient struct {
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

func newModelInferenceClient(endpoint, account, subscriptionKey string) *modelInferenceClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"ai-foundry-model-inference-2024-05-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &modelInferenceClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *modelInferenceClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
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
	model := getenv("STACKYARD_AZURE_AI_FOUNDRY_MODEL", "phi-4")

	fmt.Printf("Stackyard Azure AI Foundry Model Inference (model-inference-2024-05-01-preview) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newModelInferenceClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "GetModelInfo",
			method:   http.MethodGet,
			path:     "/azure/ai-foundry/model-inference/models/info?api-version=" + modelInferenceAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "GetEmbeddings",
			method: http.MethodPost,
			path:   "/azure/ai-foundry/model-inference/models/embeddings?api-version=" + modelInferenceAPIVersion,
			payload: map[string]any{
				"input": []string{"Stackyard Azure AI Foundry model inference embeddings example"},
				"model": model,
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "GetImageEmbeddings",
			method: http.MethodPost,
			path:   "/azure/ai-foundry/model-inference/models/images/embeddings?api-version=" + modelInferenceAPIVersion,
			payload: map[string]any{
				"input": []map[string]any{
					{"image": "aGVsbG8=", "text": "sample image"},
				},
				"model": model,
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "GetChatCompletions",
			method: http.MethodPost,
			path:   "/azure/ai-foundry/model-inference/models/chat/completions?api-version=" + modelInferenceAPIVersion,
			payload: map[string]any{
				"messages": []map[string]any{
					{"role": "system", "content": "You are a concise assistant."},
					{"role": "user", "content": "Say hello from Stackyard"},
				},
				"model": model,
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
		fmt.Println("All model-inference routes are staged in this Stackyard build.")
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
