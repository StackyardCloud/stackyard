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

const blueprintsAPIVersion = "2018-11-01-preview"

type blueprintsClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type blueprintsSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *blueprintsSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newBlueprintsClient(endpoint, account, subscriptionKey string) *blueprintsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"blueprints-2018-11-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&blueprintsSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &blueprintsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *blueprintsClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return map[string]any{}, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, nil
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_BLUEPRINTS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_BLUEPRINTS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	managementGroup := getenv("STACKYARD_AZURE_MANAGEMENT_GROUP", "contoso")

	client := newBlueprintsClient(endpoint, account, subscriptionKey)
	mgBase := "/azure/providers/Microsoft.Management/managementGroups/" + managementGroup + "/providers/Microsoft.Blueprint"
	subBase := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.Blueprint"
	blueprintVersionID := "/providers/Microsoft.Management/managementGroups/" + managementGroup + "/providers/Microsoft.Blueprint/blueprints/platform/versions/v1"

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "CreateBlueprintDefinition",
			method: http.MethodPut,
			path:   mgBase + "/blueprints/platform?api-version=" + blueprintsAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": "Platform Blueprint",
					"targetScope": "subscription",
					"parameters": map[string]any{
						"location": map[string]any{"type": "string"},
					},
					"resourceGroups": map[string]any{
						"core": map[string]any{"name": "rg-core", "location": "eastus"},
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateDraftArtifact",
			method: http.MethodPut,
			path:   mgBase + "/blueprints/platform/artifacts/storage-account?api-version=" + blueprintsAPIVersion,
			payload: map[string]any{
				"kind": "template",
				"properties": map[string]any{
					"displayName": "Storage Account",
					"template": map[string]any{
						"resources": []any{
							map[string]any{"type": "Microsoft.Storage/storageAccounts", "name": "stackyardbp"},
						},
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "PublishBlueprintVersion",
			method: http.MethodPut,
			path:   mgBase + "/blueprints/platform/versions/v1?api-version=" + blueprintsAPIVersion,
			payload: map[string]any{
				"displayName": "Platform Blueprint v1",
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListPublishedArtifacts",
			method:   http.MethodGet,
			path:     mgBase + "/blueprints/platform/versions/v1/artifacts?api-version=" + blueprintsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateAssignment",
			method: http.MethodPut,
			path:   subBase + "/blueprintAssignments/assignment-a?api-version=" + blueprintsAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"identity": map[string]any{"type": "SystemAssigned"},
				"properties": map[string]any{
					"blueprintId": blueprintVersionID,
					"locks":       map[string]any{"mode": "AllResourcesDoNotDelete"},
					"parameters": map[string]any{
						"location": map[string]any{"value": "eastus"},
					},
					"resourceGroups": map[string]any{
						"core": map[string]any{"name": "rg-core", "location": "eastus"},
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListAssignmentOperations",
			method:   http.MethodGet,
			path:     subBase + "/blueprintAssignments/assignment-a/assignmentOperations?api-version=" + blueprintsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "WhoIsBlueprint",
			method:   http.MethodPost,
			path:     subBase + "/blueprintAssignments/assignment-a/whoIsBlueprint?api-version=" + blueprintsAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
	}

	fmt.Printf("Stackyard Azure Blueprints example using %s\n", strings.TrimRight(endpoint, "/"))
	for _, call := range calls {
		payload, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s ok\n", call.name)
		if len(payload) > 0 {
			pretty, err := json.MarshalIndent(payload, "", "  ")
			if err == nil {
				fmt.Println(string(pretty))
			}
		}
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
