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

const openAIAPIVersion = "2024-10-21"

type openAIClient struct {
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

func newOpenAIClient(endpoint, account, subscriptionKey string) *openAIClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"openai-2024-10-21",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &openAIClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *openAIClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

	fileID := getenv("STACKYARD_AZURE_OPENAI_FILE_ID", "file-abc")
	batchID := getenv("STACKYARD_AZURE_OPENAI_BATCH_ID", "batch-123")
	fineTuningJobID := getenv("STACKYARD_AZURE_OPENAI_FINE_TUNING_JOB_ID", "ftjob-123")
	uploadID := getenv("STACKYARD_AZURE_OPENAI_UPLOAD_ID", "upload-123")
	modelName := getenv("STACKYARD_AZURE_OPENAI_MODEL_NAME", "gpt-4o-mini")

	fmt.Printf("Stackyard Azure OpenAI (openai-2024-10-21) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newOpenAIClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "UploadFile",
			method: http.MethodPost,
			path:   "/azure/openai/files?api-version=" + openAIAPIVersion,
			payload: map[string]any{
				"purpose": "assistants",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetFile",
			method:   http.MethodGet,
			path:     "/azure/openai/files/" + fileID + "?api-version=" + openAIAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateBatch",
			method: http.MethodPost,
			path:   "/azure/openai/batches?api-version=" + openAIAPIVersion,
			payload: map[string]any{
				"input_file_id":      fileID,
				"endpoint":           "/chat/completions",
				"completion_window":  "24h",
				"metadata":           map[string]any{"env": "local"},
				"additional_details": map[string]any{},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetBatch",
			method:   http.MethodGet,
			path:     "/azure/openai/batches/" + batchID + "?api-version=" + openAIAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "CancelBatch",
			method:   http.MethodPost,
			path:     "/azure/openai/batches/" + batchID + "/cancel?api-version=" + openAIAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateFineTuningJob",
			method: http.MethodPost,
			path:   "/azure/openai/fine_tuning/jobs?api-version=" + openAIAPIVersion,
			payload: map[string]any{
				"training_file": fileID,
				"model":         modelName,
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListFineTuningEvents",
			method:   http.MethodGet,
			path:     "/azure/openai/fine_tuning/jobs/" + fineTuningJobID + "/events?api-version=" + openAIAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListModels",
			method:   http.MethodGet,
			path:     "/azure/openai/models?api-version=" + openAIAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetModel",
			method:   http.MethodGet,
			path:     "/azure/openai/models/" + modelName + "?api-version=" + openAIAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "AddUploadPart",
			method: http.MethodPost,
			path:   "/azure/openai/uploads/" + uploadID + "/parts?api-version=" + openAIAPIVersion,
			payload: map[string]any{
				"content": "cGFydA==",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CompleteUpload",
			method: http.MethodPost,
			path:   "/azure/openai/uploads/" + uploadID + "/complete?api-version=" + openAIAPIVersion,
			payload: map[string]any{
				"part_ids": []string{"part-1", "part-2"},
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
		fmt.Println("All openai routes are staged in this Stackyard build.")
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
