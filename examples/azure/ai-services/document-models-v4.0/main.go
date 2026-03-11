package main

import (
	"bytes"
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

const documentModelsAPIVersion = "2024-11-30"

type documentModelsClient struct {
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

func newDocumentModelsClient(endpoint, account, subscriptionKey string) *documentModelsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"ai-services-data-plane-document-models-v4.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &documentModelsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *documentModelsClient) doRequest(ctx context.Context, method, path string, payload any, contentType string, expectedStatuses ...int) ([]byte, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

	if payload != nil {
		if body, ok := payload.([]byte); ok {
			if strings.TrimSpace(contentType) == "" {
				contentType = "application/octet-stream"
			}
			req.Raw().Header.Set("Content-Type", contentType)
			req.Raw().Body = io.NopCloser(bytes.NewReader(body))
			req.Raw().ContentLength = int64(len(body))
		} else {
			if err := runtime.MarshalAsJSON(req, payload); err != nil {
				return nil, 0, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
			}
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
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, resp.StatusCode, nil
		}
		return body, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	for _, status := range expectedStatuses {
		if resp.StatusCode == status {
			return body, resp.StatusCode, nil
		}
	}
	return body, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func (c *documentModelsClient) doJSON(ctx context.Context, method, path string, payload any, contentType string, expectedStatuses ...int) (map[string]any, int, error) {
	body, status, err := c.doRequest(ctx, method, path, payload, contentType, expectedStatuses...)
	if err != nil {
		return nil, status, err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return map[string]any{}, status, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, status, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, status, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	modelID := getenv("STACKYARD_AZURE_AISERVICES_MODEL_ID", "custom-model")
	targetModelID := getenv("STACKYARD_AZURE_AISERVICES_TARGET_MODEL_ID", modelID+"-copy")
	resultID := getenv("STACKYARD_AZURE_AISERVICES_RESULT_ID", "result-1")
	batchResultID := getenv("STACKYARD_AZURE_AISERVICES_BATCH_RESULT_ID", "batch-result-1")
	figureID := getenv("STACKYARD_AZURE_AISERVICES_FIGURE_ID", "1.1")

	fmt.Printf("Stackyard Azure AI Services - Data Plane - Document Models (ai-services-data-plane-document-models-v4.0) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newDocumentModelsClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name      string
		method    string
		path      string
		payload   any
		mime      string
		statuses  []int
		expectRaw bool
	}{
		{
			name:   "AnalyzeBatchDocuments",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels/" + modelID + ":analyzeBatch",
			payload: map[string]any{
				"azureBlobSource": map[string]any{
					"containerUrl": "https://example.blob.core.windows.net/source",
				},
				"resultContainerUrl": "https://example.blob.core.windows.net/results",
			},
			statuses: []int{http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:   "AnalyzeDocument",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels/" + modelID + ":analyze?_overload=analyzeDocument",
			payload: map[string]any{
				"urlSource": "https://example.com/invoice.pdf",
			},
			statuses: []int{http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:     "AnalyzeDocumentFromStream",
			method:   http.MethodPost,
			path:     "/azure/aiservices/documentintelligence/documentModels/" + modelID + ":analyze",
			payload:  []byte("%PDF-1.7 stream-content"),
			mime:     "application/pdf",
			statuses: []int{http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:   "AuthorizeModelCopy",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels:authorizeCopy",
			payload: map[string]any{
				"modelId":     targetModelID,
				"description": "stackyard local copy authorization",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "BuildModel",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels:build",
			payload: map[string]any{
				"modelId":   modelID,
				"buildMode": "template",
				"azureBlobSource": map[string]any{
					"containerUrl": "https://example.blob.core.windows.net/source",
				},
			},
			statuses: []int{http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:   "ComposeModel",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels:compose",
			payload: map[string]any{
				"modelId": targetModelID,
				"componentModels": []map[string]any{
					{"modelId": modelID + "-a"},
					{"modelId": modelID + "-b"},
				},
			},
			statuses: []int{http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:   "CopyModelTo",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels/" + modelID + ":copyTo",
			payload: map[string]any{
				"targetResourceId":     "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.CognitiveServices/accounts/stackyard-target",
				"targetResourceRegion": "eastus",
				"targetModelId":        targetModelID,
				"accessToken":          "stackyard-copy-token",
				"expirationDateTime":   "2099-01-01T00:00:00Z",
			},
			statuses: []int{http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:     "DeleteAnalyzeBatchResult",
			method:   http.MethodDelete,
			path:     "/azure/aiservices/documentintelligence/documentModels/" + modelID + "/analyzeBatchResults/" + batchResultID,
			statuses: []int{http.StatusNoContent, http.StatusNotFound, http.StatusNotImplemented},
		},
		{
			name:     "DeleteAnalyzeResult",
			method:   http.MethodDelete,
			path:     "/azure/aiservices/documentintelligence/documentModels/" + modelID + "/analyzeResults/" + resultID,
			statuses: []int{http.StatusNoContent, http.StatusNotFound, http.StatusNotImplemented},
		},
		{
			name:     "DeleteModel",
			method:   http.MethodDelete,
			path:     "/azure/aiservices/documentintelligence/documentModels/" + modelID,
			statuses: []int{http.StatusNoContent, http.StatusNotFound, http.StatusNotImplemented},
		},
		{
			name:     "GetAnalyzeBatchResult",
			method:   http.MethodGet,
			path:     "/azure/aiservices/documentintelligence/documentModels/" + modelID + "/analyzeBatchResults/" + batchResultID,
			statuses: []int{http.StatusOK, http.StatusNotFound, http.StatusNotImplemented},
		},
		{
			name:     "GetAnalyzeResult",
			method:   http.MethodGet,
			path:     "/azure/aiservices/documentintelligence/documentModels/" + modelID + "/analyzeResults/" + resultID,
			statuses: []int{http.StatusOK, http.StatusNotFound, http.StatusNotImplemented},
		},
		{
			name:      "GetAnalyzeResultFigure",
			method:    http.MethodGet,
			path:      "/azure/aiservices/documentintelligence/documentModels/" + modelID + "/analyzeResults/" + resultID + "/figures/" + figureID,
			statuses:  []int{http.StatusOK, http.StatusNotFound, http.StatusNotImplemented},
			expectRaw: true,
		},
		{
			name:      "GetAnalyzeResultPDF",
			method:    http.MethodGet,
			path:      "/azure/aiservices/documentintelligence/documentModels/" + modelID + "/analyzeResults/" + resultID + "/pdf",
			statuses:  []int{http.StatusOK, http.StatusNotFound, http.StatusNotImplemented},
			expectRaw: true,
		},
		{
			name:     "GetModel",
			method:   http.MethodGet,
			path:     "/azure/aiservices/documentintelligence/documentModels/" + modelID,
			statuses: []int{http.StatusOK, http.StatusNotFound, http.StatusNotImplemented},
		},
		{
			name:     "ListAnalyzeBatchResults",
			method:   http.MethodGet,
			path:     "/azure/aiservices/documentintelligence/documentModels/" + modelID + "/analyzeBatchResults",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListModels",
			method:   http.MethodGet,
			path:     "/azure/aiservices/documentintelligence/documentModels",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		fullPath := withAPIVersion(call.path, documentModelsAPIVersion)
		if call.expectRaw {
			_, status, err := client.doRequest(ctx, call.method, fullPath, call.payload, call.mime, call.statuses...)
			if err != nil {
				exitf("%s failed: %v", call.name, err)
			}
			if status == http.StatusNotImplemented {
				notImplementedCount++
				fmt.Printf("%s: route recognized but not implemented yet (%s)\n", call.name, fullPath)
				continue
			}
			fmt.Printf("%s: status=%d\n", call.name, status)
			continue
		}

		_, status, err := client.doJSON(ctx, call.method, fullPath, call.payload, call.mime, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		if status == http.StatusNotImplemented {
			notImplementedCount++
			fmt.Printf("%s: route recognized but not implemented yet (%s)\n", call.name, fullPath)
			continue
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All document model routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
}

func withAPIVersion(path, apiVersion string) string {
	if strings.Contains(path, "?") {
		return path + "&api-version=" + apiVersion
	}
	return path + "?api-version=" + apiVersion
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
