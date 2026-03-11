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

const speechToTextAPIVersion = "2024-11-01"

type speechToTextClient struct {
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

func newSpeechToTextClient(endpoint, account, subscriptionKey string) *speechToTextClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"speech-to-text-v3.2-preview.2",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &speechToTextClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *speechToTextClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	locale := getenv("STACKYARD_AZURE_SPEECH_TO_TEXT_LOCALE", "en-US")
	transcriptionID := getenv("STACKYARD_AZURE_SPEECH_TO_TEXT_TRANSCRIPTION_ID", "transcription-a")
	projectID := getenv("STACKYARD_AZURE_SPEECH_TO_TEXT_PROJECT_ID", "project-a")
	webhookID := getenv("STACKYARD_AZURE_SPEECH_TO_TEXT_WEBHOOK_ID", "webhook-a")

	fmt.Printf("Stackyard Azure Speech to Text (speech-to-text-v3.2-preview.2) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newSpeechToTextClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "CreateDataset",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets?api-version=" + speechToTextAPIVersion,
			payload: map[string]any{
				"displayName": "stackyard-dataset",
				"locale":      locale,
				"kind":        "Acoustic",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListEndpoints",
			method:   http.MethodGet,
			path:     "/azure/speechtotext/v3.2-preview.2/endpoints?api-version=" + speechToTextAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetBaseModel",
			method:   http.MethodGet,
			path:     "/azure/speechtotext/v3.2-preview.2/models/base/" + locale + "?api-version=" + speechToTextAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetServiceHealth",
			method:   http.MethodGet,
			path:     "/azure/speechtotext/v3.2-preview.2/healthstatus?api-version=" + speechToTextAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateProject",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/projects?api-version=" + speechToTextAPIVersion,
			payload: map[string]any{
				"displayName": "stackyard-project",
				"locale":      locale,
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateTranscription",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/transcriptions?api-version=" + speechToTextAPIVersion,
			payload: map[string]any{
				"displayName": transcriptionID,
				"locale":      locale,
				"contentUrls": []string{"https://example.com/audio.wav"},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListTranscriptions",
			method:   http.MethodGet,
			path:     "/azure/speechtotext/v3.2-preview.2/transcriptions?api-version=" + speechToTextAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetOperation",
			method:   http.MethodGet,
			path:     "/azure/speechtotext/v3.2-preview.2/operations/op-123?api-version=" + speechToTextAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "PingWebhook",
			method:   http.MethodPost,
			path:     "/azure/speechtotext/v3.2-preview.2/webhooks/" + webhookID + "/ping?api-version=" + speechToTextAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListProjectModels",
			method:   http.MethodGet,
			path:     "/azure/speechtotext/v3.2-preview.2/projects/" + projectID + "/models?api-version=" + speechToTextAPIVersion,
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
		fmt.Println("All speech-to-text routes are staged in this Stackyard build.")
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
