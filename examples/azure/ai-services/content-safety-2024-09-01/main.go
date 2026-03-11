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

const contentSafetyAPIVersion = "2024-09-01"

type contentSafetyClient struct {
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

func newContentSafetyClient(endpoint, account, subscriptionKey string) *contentSafetyClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"content-safety-2024-09-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &contentSafetyClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *contentSafetyClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_CONTENT_SAFETY_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CONTENT_SAFETY_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	blocklistName := getenv("STACKYARD_AZURE_CONTENT_SAFETY_BLOCKLIST", "local-list")
	blocklistItemID := getenv("STACKYARD_AZURE_CONTENT_SAFETY_BLOCKLIST_ITEM_ID", "item-1")
	imageURL := getenv("STACKYARD_AZURE_CONTENT_SAFETY_IMAGE_URL", "https://example.com/safe-image.jpg")
	textSample := getenv("STACKYARD_AZURE_CONTENT_SAFETY_TEXT", "hello world")

	fmt.Printf("Stackyard Azure Content Safety (content-safety-2024-09-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newContentSafetyClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "AnalyzeImage",
			method: http.MethodPost,
			path:   "/azure/contentsafety/image:analyze?api-version=" + contentSafetyAPIVersion,
			payload: map[string]any{
				"image": map[string]any{
					"url": imageURL,
				},
				"categories": []string{"Hate", "Sexual"},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "AnalyzeText",
			method: http.MethodPost,
			path:   "/azure/contentsafety/text:analyze?api-version=" + contentSafetyAPIVersion,
			payload: map[string]any{
				"text":       textSample,
				"categories": []string{"Hate", "Sexual"},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "DetectProtectedMaterial",
			method: http.MethodPost,
			path:   "/azure/contentsafety/text:detectProtectedMaterial?api-version=" + contentSafetyAPIVersion,
			payload: map[string]any{
				"text": textSample,
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "ShieldPrompt",
			method: http.MethodPost,
			path:   "/azure/contentsafety/text:shieldPrompt?api-version=" + contentSafetyAPIVersion,
			payload: map[string]any{
				"userPrompt": "Summarize the provided document.",
				"documents":  []string{"Document content for prompt shielding."},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListTextBlocklists",
			method:   http.MethodGet,
			path:     "/azure/contentsafety/text/blocklists?api-version=" + contentSafetyAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateOrUpdateTextBlocklist",
			method: http.MethodPatch,
			path:   "/azure/contentsafety/text/blocklists/" + blocklistName + "?api-version=" + contentSafetyAPIVersion,
			payload: map[string]any{
				"description": "stackyard content safety list",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetTextBlocklist",
			method:   http.MethodGet,
			path:     "/azure/contentsafety/text/blocklists/" + blocklistName + "?api-version=" + contentSafetyAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "AddOrUpdateTextBlocklistItems",
			method: http.MethodPost,
			path:   "/azure/contentsafety/text/blocklists/" + blocklistName + ":addOrUpdateBlocklistItems?api-version=" + contentSafetyAPIVersion,
			payload: map[string]any{
				"blocklistItems": []map[string]any{
					{"text": "forbidden phrase"},
				},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListTextBlocklistItems",
			method:   http.MethodGet,
			path:     "/azure/contentsafety/text/blocklists/" + blocklistName + "/blocklistItems?api-version=" + contentSafetyAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetTextBlocklistItem",
			method:   http.MethodGet,
			path:     "/azure/contentsafety/text/blocklists/" + blocklistName + "/blocklistItems/" + blocklistItemID + "?api-version=" + contentSafetyAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "RemoveTextBlocklistItems",
			method: http.MethodPost,
			path:   "/azure/contentsafety/text/blocklists/" + blocklistName + ":removeBlocklistItems?api-version=" + contentSafetyAPIVersion,
			payload: map[string]any{
				"blocklistItemIds": []string{blocklistItemID},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "DeleteTextBlocklist",
			method:   http.MethodDelete,
			path:     "/azure/contentsafety/text/blocklists/" + blocklistName + "?api-version=" + contentSafetyAPIVersion,
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
		fmt.Println("All content safety routes are staged in this Stackyard build.")
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
