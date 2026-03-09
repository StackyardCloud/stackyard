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

const miscellaneousOperationsAPIVersion = "2024-11-30"

type miscellaneousOperationsClient struct {
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

func newMiscellaneousOperationsClient(endpoint, account, subscriptionKey string) *miscellaneousOperationsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"ai-services-data-plane-miscellaneous-operations-v4.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &miscellaneousOperationsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *miscellaneousOperationsClient) doJSON(ctx context.Context, method, path string, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

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
			decoded, decodeErr := decodeResponseJSON(body)
			return decoded, resp.StatusCode, decodeErr
		}
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	for _, status := range expectedStatuses {
		if resp.StatusCode == status {
			decoded, decodeErr := decodeResponseJSON(body)
			return decoded, resp.StatusCode, decodeErr
		}
	}
	return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func decodeResponseJSON(body []byte) (map[string]any, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return map[string]any{}, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode JSON body: %w", err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	operationID := getenv("STACKYARD_AZURE_AISERVICES_OPERATION_ID", "op-123")

	fmt.Printf("Stackyard Azure AI Services - Data Plane - Miscellaneous Operations (ai-services-data-plane-miscellaneous-operations-v4.0) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newMiscellaneousOperationsClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name string
		path string
	}{
		{
			name: "GetResourceDetails",
			path: "/azure/aiservices/documentintelligence/info",
		},
		{
			name: "ListOperations",
			path: "/azure/aiservices/documentintelligence/operations",
		},
		{
			name: "GetDocumentModelBuildOperation",
			path: "/azure/aiservices/documentintelligence/operations/" + operationID,
		},
		{
			name: "GetOperation",
			path: "/azure/aiservices/documentintelligence/operations/" + operationID + "?_overload=getOperation",
		},
		{
			name: "GetDocumentModelComposeOperation",
			path: "/azure/aiservices/documentintelligence/operations/" + operationID + "?_overload=getDocumentModelComposeOperation",
		},
		{
			name: "GetDocumentModelCopyToOperation",
			path: "/azure/aiservices/documentintelligence/operations/" + operationID + "?_overload=getDocumentModelCopyToOperation",
		},
		{
			name: "GetDocumentClassifierBuildOperation",
			path: "/azure/aiservices/documentintelligence/operations/" + operationID + "?_overload=getDocumentClassifierBuildOperation",
		},
		{
			name: "GetDocumentClassifierCopyToOperation",
			path: "/azure/aiservices/documentintelligence/operations/" + operationID + "?_overload=getDocumentClassifierCopyToOperation",
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		fullPath := withAPIVersion(call.path, miscellaneousOperationsAPIVersion)
		_, status, err := client.doJSON(ctx, http.MethodGet, fullPath, http.StatusOK, http.StatusNotFound, http.StatusNotImplemented)
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
		fmt.Println("All miscellaneous operation routes are staged in this Stackyard build.")
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
