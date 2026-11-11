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

const quantumAPIVersion = "2025-12-15-preview"

type quantumClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type quantumSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *quantumSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newQuantumClient(endpoint, account, subscriptionKey string) *quantumClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"azurequantum-resourcemanager-2025-12-15-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&quantumSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &quantumClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *quantumClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_QUANTUM_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_QUANTUM_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-quantum")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	workspaceName := getenv("STACKYARD_AZURE_QUANTUM_WORKSPACE_NAME", "workspace-a")

	fmt.Printf("Stackyard Azure Quantum example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newQuantumClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	resourceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Quantum/workspaces/" + workspaceName

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
			path:     "/azure/providers/Microsoft.Quantum/operations?api-version=" + quantumAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path:   subscriptionBase + "/providers/Microsoft.Quantum/checkNameAvailability?api-version=" + quantumAPIVersion,
			payload: map[string]any{
				"name": workspaceName,
				"type": "Microsoft.Quantum/workspaces",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListOfferings",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.Quantum/locations/" + region + "/offerings?api-version=" + quantumAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateWorkspace",
			method: http.MethodPut,
			path:   resourceBase + "?api-version=" + quantumAPIVersion,
			payload: map[string]any{
				"location": region,
				"tags":     map[string]any{"env": "dev"},
				"identity": map[string]any{"type": "SystemAssigned"},
				"properties": map[string]any{
					"workspaceDescription": "Local workspace",
					"apiKeyEnabled":        true,
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetWorkspace",
			method:   http.MethodGet,
			path:     resourceBase + "?api-version=" + quantumAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListByResourceGroup",
			method:   http.MethodGet,
			path:     subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Quantum/workspaces?api-version=" + quantumAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListBySubscription",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.Quantum/workspaces?api-version=" + quantumAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListKeys",
			method:   http.MethodPost,
			path:     resourceBase + "/listKeys?api-version=" + quantumAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "RegeneratePrimaryKey",
			method: http.MethodPost,
			path:   resourceBase + "/regenerateKeys?api-version=" + quantumAPIVersion,
			payload: map[string]any{
				"keyName": "Primary",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PatchWorkspace",
			method: http.MethodPatch,
			path:   resourceBase + "?api-version=" + quantumAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"env":   "stage",
					"owner": "stackyard",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteWorkspace",
			method:   http.MethodDelete,
			path:     resourceBase + "?api-version=" + quantumAPIVersion,
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
