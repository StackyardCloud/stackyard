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

const customVoiceAPIVersion = "2024-02-01-preview"

type customVoiceClient struct {
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

func newCustomVoiceClient(endpoint, account, subscriptionKey string) *customVoiceClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"custom-voice-2024-02-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &customVoiceClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *customVoiceClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	projectID := getenv("STACKYARD_AZURE_CUSTOM_VOICE_PROJECT_ID", "project-a")
	endpointID := getenv("STACKYARD_AZURE_CUSTOM_VOICE_ENDPOINT_ID", "endpoint-a")
	modelID := getenv("STACKYARD_AZURE_CUSTOM_VOICE_MODEL_ID", "model-a")
	consentID := getenv("STACKYARD_AZURE_CUSTOM_VOICE_CONSENT_ID", "consent-a")
	personalVoiceID := getenv("STACKYARD_AZURE_CUSTOM_VOICE_PERSONAL_VOICE_ID", "personal-a")
	trainingSetID := getenv("STACKYARD_AZURE_CUSTOM_VOICE_TRAINING_SET_ID", "set-a")

	fmt.Printf("Stackyard Azure Custom Voice (custom-voice-2024-02-01-preview) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCustomVoiceClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListBaseModels",
			method:   http.MethodGet,
			path:     "/azure/customvoice/basemodels?api-version=" + customVoiceAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateProject",
			method: http.MethodPut,
			path:   "/azure/customvoice/projects/" + projectID + "?api-version=" + customVoiceAPIVersion,
			payload: map[string]any{
				"displayName": "stackyard-project",
				"locale":      "en-US",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateConsent",
			method: http.MethodPut,
			path:   "/azure/customvoice/consents/" + consentID + "?api-version=" + customVoiceAPIVersion,
			payload: map[string]any{
				"email":    "stackyard@example.com",
				"fullName": "Stackyard Local",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateTrainingSet",
			method: http.MethodPut,
			path:   "/azure/customvoice/trainingsets/" + trainingSetID + "?api-version=" + customVoiceAPIVersion,
			payload: map[string]any{
				"projectId": projectID,
				"locale":    "en-US",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "UploadTrainingSetData",
			method: http.MethodPost,
			path:   "/azure/customvoice/trainingsets/" + trainingSetID + ":upload?api-version=" + customVoiceAPIVersion,
			payload: map[string]any{
				"files": []map[string]any{
					{
						"name": "audio.wav",
						"url":  "https://example.com/audio.wav",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateModel",
			method: http.MethodPut,
			path:   "/azure/customvoice/models/" + modelID + "?api-version=" + customVoiceAPIVersion,
			payload: map[string]any{
				"projectId":      projectID,
				"trainingSetIds": []string{trainingSetID},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListModelRecipes",
			method:   http.MethodGet,
			path:     "/azure/customvoice/modelrecipes?api-version=" + customVoiceAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateEndpoint",
			method: http.MethodPut,
			path:   "/azure/customvoice/endpoints/" + endpointID + "?api-version=" + customVoiceAPIVersion,
			payload: map[string]any{
				"projectId": projectID,
				"modelId":   modelID,
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ResumeEndpoint",
			method:   http.MethodPost,
			path:     "/azure/customvoice/endpoints/" + endpointID + ":resume?api-version=" + customVoiceAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreatePersonalVoice",
			method: http.MethodPut,
			path:   "/azure/customvoice/personalvoices/" + personalVoiceID + "?api-version=" + customVoiceAPIVersion,
			payload: map[string]any{
				"consentId": consentID,
				"projectId": projectID,
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetOperation",
			method:   http.MethodGet,
			path:     "/azure/customvoice/operations/op-123?api-version=" + customVoiceAPIVersion,
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
		fmt.Println("All custom voice routes are staged in this Stackyard build.")
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
