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

const contentUnderstandingAPIVersion = "2025-11-01"

type contentUnderstandingClient struct {
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

func newContentUnderstandingClient(endpoint, account, subscriptionKey string) *contentUnderstandingClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"content-understanding-2025-11-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &contentUnderstandingClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *contentUnderstandingClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_CONTENT_UNDERSTANDING_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CONTENT_UNDERSTANDING_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	analyzerID := getenv("STACKYARD_AZURE_CONTENT_UNDERSTANDING_ANALYZER_ID", "analyzer-a")
	operationID := getenv("STACKYARD_AZURE_CONTENT_UNDERSTANDING_OPERATION_ID", "op-1")
	fileID := getenv("STACKYARD_AZURE_CONTENT_UNDERSTANDING_FILE_ID", "file-1")

	fmt.Printf("Stackyard Azure Content Understanding (content-understanding-2025-11-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newContentUnderstandingClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListAnalyzers",
			method:   http.MethodGet,
			path:     "/azure/contentunderstanding/analyzers?api-version=" + contentUnderstandingAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateOrReplaceAnalyzer",
			method: http.MethodPut,
			path:   "/azure/contentunderstanding/analyzers/" + analyzerID + "?api-version=" + contentUnderstandingAPIVersion,
			payload: map[string]any{
				"description": "stackyard analyzer",
				"scenario":    "generalDocument",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "UpdateAnalyzer",
			method: http.MethodPatch,
			path:   "/azure/contentunderstanding/analyzers/" + analyzerID + "?api-version=" + contentUnderstandingAPIVersion,
			payload: map[string]any{
				"description": "updated stackyard analyzer",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetAnalyzer",
			method:   http.MethodGet,
			path:     "/azure/contentunderstanding/analyzers/" + analyzerID + "?api-version=" + contentUnderstandingAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "Analyze",
			method: http.MethodPost,
			path:   "/azure/contentunderstanding/analyzers/" + analyzerID + ":analyze?api-version=" + contentUnderstandingAPIVersion,
			payload: map[string]any{
				"documentLocation": "https://example.com/invoice.pdf",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "AnalyzeBinary",
			method: http.MethodPost,
			path:   "/azure/contentunderstanding/analyzers/" + analyzerID + ":analyzeBinary?api-version=" + contentUnderstandingAPIVersion,
			payload: map[string]any{
				"contentType": "application/pdf",
				"content":     "JVBERi0xLjQKJ...",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "GrantCopyAuthorization",
			method: http.MethodPost,
			path:   "/azure/contentunderstanding/analyzers/" + analyzerID + ":grantCopyAuthorization?api-version=" + contentUnderstandingAPIVersion,
			payload: map[string]any{
				"targetRegion":     "eastus",
				"targetAnalyzerId": "analyzer-target",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CopyAnalyzer",
			method: http.MethodPost,
			path:   "/azure/contentunderstanding/analyzers/" + analyzerID + ":copy?api-version=" + contentUnderstandingAPIVersion,
			payload: map[string]any{
				"targetAnalyzerResourceId": "/subscriptions/000/resourceGroups/rg/providers/Microsoft.CognitiveServices/accounts/ai/analyzers/analyzer-target",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetOperationStatus",
			method:   http.MethodGet,
			path:     "/azure/contentunderstanding/analyzers/" + analyzerID + "/operations/" + operationID + "?api-version=" + contentUnderstandingAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetResult",
			method:   http.MethodGet,
			path:     "/azure/contentunderstanding/analyzerResults/" + operationID + "?api-version=" + contentUnderstandingAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetResultFile",
			method:   http.MethodGet,
			path:     "/azure/contentunderstanding/analyzerResults/" + operationID + "/files/" + fileID + "?api-version=" + contentUnderstandingAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "DeleteResult",
			method:   http.MethodDelete,
			path:     "/azure/contentunderstanding/analyzerResults/" + operationID + "?api-version=" + contentUnderstandingAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetDefaults",
			method:   http.MethodGet,
			path:     "/azure/contentunderstanding/defaults?api-version=" + contentUnderstandingAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "UpdateDefaults",
			method: http.MethodPatch,
			path:   "/azure/contentunderstanding/defaults?api-version=" + contentUnderstandingAPIVersion,
			payload: map[string]any{
				"defaultLocale": "en-US",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "DeleteAnalyzer",
			method:   http.MethodDelete,
			path:     "/azure/contentunderstanding/analyzers/" + analyzerID + "?api-version=" + contentUnderstandingAPIVersion,
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
		fmt.Println("All content understanding routes are staged in this Stackyard build.")
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
