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

const playwrightAPIVersion = "2025-09-01"

type playwrightClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type playwrightSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *playwrightSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newPlaywrightClient(endpoint, account, subscriptionKey string) *playwrightClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"playwright-resourcemanager-2025-09-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&playwrightSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &playwrightClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *playwrightClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, readErr)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	if len(expectedStatuses) > 0 {
		for _, expected := range expectedStatuses {
			if status == expected {
				return nil
			}
		}
		return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_PLAYWRIGHT_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_PLAYWRIGHT_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-playwright")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	workspaceName := getenv("STACKYARD_AZURE_PLAYWRIGHT_WORKSPACE_NAME", "workspace-a")

	fmt.Printf("Stackyard Azure Playwright example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newPlaywrightClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	resourceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/playwrightWorkspaces/" + workspaceName

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.LoadTestService/operations?api-version=" + playwrightAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path:   subscriptionBase + "/providers/Microsoft.LoadTestService/checkNameAvailability?api-version=" + playwrightAPIVersion,
			payload: map[string]any{
				"name": workspaceName,
				"type": "Microsoft.LoadTestService/playwrightWorkspaces",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateWorkspace",
			method: http.MethodPut,
			path:   resourceBase + "?api-version=" + playwrightAPIVersion,
			payload: map[string]any{
				"location": region,
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"localAuth":        "Enabled",
					"regionalAffinity": "Enabled",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetWorkspace",
			method:   http.MethodGet,
			path:     resourceBase + "?api-version=" + playwrightAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListByResourceGroup",
			method:   http.MethodGet,
			path:     subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/playwrightWorkspaces?api-version=" + playwrightAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListBySubscription",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.LoadTestService/playwrightWorkspaces?api-version=" + playwrightAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PatchWorkspace",
			method: http.MethodPatch,
			path:   resourceBase + "?api-version=" + playwrightAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"env":   "stage",
					"owner": "stackyard",
				},
				"properties": map[string]any{
					"localAuth": "Disabled",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSubscriptionQuotas",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.LoadTestService/locations/" + region + "/playwrightQuotas?api-version=" + playwrightAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetSubscriptionQuota",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.LoadTestService/locations/" + region + "/playwrightQuotas/ExecutionMinutes?api-version=" + playwrightAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListWorkspaceQuotas",
			method:   http.MethodGet,
			path:     resourceBase + "/quotas?api-version=" + playwrightAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetWorkspaceQuota",
			method:   http.MethodGet,
			path:     resourceBase + "/quotas/ExecutionMinutes?api-version=" + playwrightAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteWorkspace",
			method:   http.MethodDelete,
			path:     resourceBase + "?api-version=" + playwrightAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}
}

func printJSON(name string, status int, payload map[string]any) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("marshal %s response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n%s\n", name, status, string(body))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
