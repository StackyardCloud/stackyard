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

const videoTranslationAPIVersion = "2026-03-01"

type videoTranslationClient struct {
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

func newVideoTranslationClient(endpoint, account, subscriptionKey string) *videoTranslationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"video-translation-2026-03-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &videoTranslationClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *videoTranslationClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	translationID := getenv("STACKYARD_AZURE_VIDEO_TRANSLATION_ID", "translation-a")
	iterationID := getenv("STACKYARD_AZURE_VIDEO_TRANSLATION_ITERATION_ID", "iteration-a")
	operationID := getenv("STACKYARD_AZURE_VIDEO_TRANSLATION_OPERATION_ID", "operation-a")

	fmt.Printf("Stackyard Azure Video Translation (video-translation-2026-03-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newVideoTranslationClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "CreateEventHubConfiguration",
			method: http.MethodPut,
			path:   "/azure/videotranslation/configurations/event-hub?api-version=" + videoTranslationAPIVersion,
			payload: map[string]any{
				"name":             "stackyard-event-hub",
				"connectionString": "Endpoint=sb://stackyard.servicebus.windows.net/",
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusNotImplemented},
		},
		{
			name:     "GetEventHubConfiguration",
			method:   http.MethodGet,
			path:     "/azure/videotranslation/configurations/event-hub?api-version=" + videoTranslationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "PingEventHubConfiguration",
			method:   http.MethodPost,
			path:     "/azure/videotranslation/configurations/event-hub:ping?api-version=" + videoTranslationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:   "CreateTranslation",
			method: http.MethodPut,
			path:   "/azure/videotranslation/translations/" + translationID + "?api-version=" + videoTranslationAPIVersion,
			payload: map[string]any{
				"displayName":  "stackyard-translation",
				"sourceLocale": "en-US",
				"targetLocales": []string{
					"es-ES",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusNotImplemented},
		},
		{
			name:     "GetTranslation",
			method:   http.MethodGet,
			path:     "/azure/videotranslation/translations/" + translationID + "?api-version=" + videoTranslationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListTranslations",
			method:   http.MethodGet,
			path:     "/azure/videotranslation/translations?api-version=" + videoTranslationAPIVersion + "&maxpagesize=20",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateTranslationIteration",
			method: http.MethodPut,
			path:   "/azure/videotranslation/translations/" + translationID + "/iterations/" + iterationID + "?api-version=" + videoTranslationAPIVersion,
			payload: map[string]any{
				"inputVideoUrl": "https://example.com/video.mp4",
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusNotImplemented},
		},
		{
			name:     "GetTranslationIteration",
			method:   http.MethodGet,
			path:     "/azure/videotranslation/translations/" + translationID + "/iterations/" + iterationID + "?api-version=" + videoTranslationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListTranslationIterations",
			method:   http.MethodGet,
			path:     "/azure/videotranslation/translations/" + translationID + "/iterations?api-version=" + videoTranslationAPIVersion + "&top=10",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetOperation",
			method:   http.MethodGet,
			path:     "/azure/videotranslation/operations/" + operationID + "?api-version=" + videoTranslationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "DeleteTranslation",
			method:   http.MethodDelete,
			path:     "/azure/videotranslation/translations/" + translationID + "?api-version=" + videoTranslationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent, http.StatusNotImplemented},
		},
		{
			name:     "DeleteEventHubConfiguration",
			method:   http.MethodDelete,
			path:     "/azure/videotranslation/configurations/event-hub?api-version=" + videoTranslationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent, http.StatusNotImplemented},
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
		fmt.Println("All video-translation routes are staged in this Stackyard build.")
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
