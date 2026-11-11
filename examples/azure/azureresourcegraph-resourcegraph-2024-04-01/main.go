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

const resourceGraphAPIVersion = "2024-04-01"
const resourceGraphHistoryAPIVersion = "2021-06-01-preview"
const resourceGraphChangesAPIVersion = "2020-09-01-preview"

type resourceGraphClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type resourceGraphSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *resourceGraphSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newResourceGraphClient(endpoint, account, subscriptionKey string) *resourceGraphClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"azureresourcegraph-resourcegraph-2024-04-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&resourceGraphSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &resourceGraphClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *resourceGraphClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_RESOURCE_GRAPH_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_RESOURCE_GRAPH_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-graph")
	queryName := getenv("STACKYARD_AZURE_RESOURCE_GRAPH_QUERY_NAME", "query-a")

	fmt.Printf("Stackyard Azure Resource Graph example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newResourceGraphClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	queryResourceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ResourceGraph/queries/" + queryName

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
			path:     "/azure/providers/Microsoft.ResourceGraph/operations?api-version=" + resourceGraphAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ResourcesQuery",
			method: http.MethodPost,
			path:   "/azure/providers/Microsoft.ResourceGraph/resources?api-version=" + resourceGraphAPIVersion,
			payload: map[string]any{
				"subscriptions": []any{subscriptionID},
				"query":         "Resources | project id, name, type",
				"options":       map[string]any{"$top": 1},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateGraphQuery",
			method: http.MethodPut,
			path:   queryResourceBase + "?api-version=" + resourceGraphAPIVersion,
			payload: map[string]any{
				"location": "global",
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"query":       "Resources | project id, name, type",
					"description": "Graph query",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetGraphQuery",
			method:   http.MethodGet,
			path:     queryResourceBase + "?api-version=" + resourceGraphAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListGraphQueriesByResourceGroup",
			method:   http.MethodGet,
			path:     subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ResourceGraph/queries?api-version=" + resourceGraphAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PatchGraphQuery",
			method: http.MethodPatch,
			path:   queryResourceBase + "?api-version=" + resourceGraphAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{"env": "stage", "owner": "stackyard"},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ResourcesHistory",
			method: http.MethodPost,
			path:   "/azure/providers/Microsoft.ResourceGraph/resourcesHistory?api-version=" + resourceGraphHistoryAPIVersion,
			payload: map[string]any{
				"subscriptions": []any{subscriptionID},
				"query":         "Resources | project id, name, type",
				"options":       map[string]any{"$top": 2},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ResourceChanges",
			method: http.MethodPost,
			path:   "/azure/providers/Microsoft.ResourceGraph/resourceChanges?api-version=" + resourceGraphChangesAPIVersion,
			payload: map[string]any{
				"subscriptions":        []any{subscriptionID},
				"interval":             map[string]any{"start": "2026-03-01T00:00:00Z", "end": "2026-04-01T00:00:00Z"},
				"$top":                 1,
				"fetchPropertyChanges": true,
				"fetchSnapshots":       true,
			},
			statuses: []int{http.StatusOK},
		},
	}

	var lastChanges map[string]any
	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		if call.name == "ResourceChanges" {
			lastChanges = payload
		}
		printJSON(call.name, status, payload)
	}

	changeID := extractFirstChangeID(lastChanges)
	if changeID == "" {
		fatalf("ResourceChanges did not return a changeId")
	}

	detailsPayload, status, err := client.doJSON(
		ctx,
		http.MethodPost,
		"/azure/providers/Microsoft.ResourceGraph/resourceChangeDetails?api-version="+resourceGraphChangesAPIVersion,
		map[string]any{
			"changeIds":   []any{changeID},
			"resourceIds": []any{"/subscriptions/" + subscriptionID + "/resourceGroups/rg-app/providers/Microsoft.Compute/virtualMachines/vm-a"},
		},
		http.StatusOK,
	)
	if err != nil {
		fatalf("ResourceChangeDetails failed: %v", err)
	}
	printJSON("ResourceChangeDetails", status, detailsPayload)

	deletePayload, status, err := client.doJSON(ctx, http.MethodDelete, queryResourceBase+"?api-version="+resourceGraphAPIVersion, nil, http.StatusOK, http.StatusNoContent)
	if err != nil {
		fatalf("DeleteGraphQuery failed: %v", err)
	}
	printJSON("DeleteGraphQuery", status, deletePayload)
}

func extractFirstChangeID(payload map[string]any) string {
	changes, _ := payload["changes"].([]any)
	if len(changes) == 0 {
		return ""
	}
	first, _ := changes[0].(map[string]any)
	if first == nil {
		return ""
	}
	id, _ := first["changeId"].(string)
	return strings.TrimSpace(id)
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
