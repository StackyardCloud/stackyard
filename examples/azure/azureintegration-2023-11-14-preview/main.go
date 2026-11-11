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

const integrationAPIVersion = "2023-11-14-preview"

type integrationEnvironmentClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type integrationSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *integrationSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newIntegrationEnvironmentClient(endpoint, account, subscriptionKey string) *integrationEnvironmentClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"azureintegration-2023-11-14-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&integrationSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &integrationEnvironmentClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *integrationEnvironmentClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_INTEGRATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_INTEGRATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-integration")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	spaceName := getenv("STACKYARD_AZURE_INTEGRATION_SPACE_NAME", "space-a")
	applicationName := getenv("STACKYARD_AZURE_INTEGRATION_APPLICATION_NAME", "app-a")
	resourceName := getenv("STACKYARD_AZURE_INTEGRATION_RESOURCE_NAME", "resource-a")
	processName := getenv("STACKYARD_AZURE_INTEGRATION_PROCESS_NAME", "process-a")

	fmt.Printf("Stackyard Azure Integration Environment example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newIntegrationEnvironmentClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	spaceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.IntegrationSpaces/spaces/" + spaceName
	applicationBase := spaceBase + "/applications/" + applicationName

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
			path:     "/azure/providers/Microsoft.IntegrationSpaces/operations?api-version=" + integrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateSpace",
			method: http.MethodPut,
			path:   spaceBase + "?api-version=" + integrationAPIVersion,
			payload: map[string]any{
				"location": region,
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"description": "example integration space",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateApplication",
			method: http.MethodPut,
			path:   applicationBase + "?api-version=" + integrationAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{"tier": "core"},
				"properties": map[string]any{
					"description": "example application",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateApplicationResource",
			method: http.MethodPut,
			path:   applicationBase + "/resources/" + resourceName + "?api-version=" + integrationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"resourceId":   "/subscriptions/mock/resourceGroups/mock/providers/Microsoft.ServiceBus/namespaces/ns1",
					"resourceType": "Microsoft.ServiceBus/namespaces",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateBusinessProcess",
			method: http.MethodPut,
			path:   applicationBase + "/businessProcesses/" + processName + "?api-version=" + integrationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"tableName":  "orders",
					"identifier": "orderId",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListBusinessProcessVersions",
			method:   http.MethodGet,
			path:     applicationBase + "/businessProcesses/" + processName + "/versions?api-version=" + integrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "SaveArtifact",
			method: http.MethodPost,
			path:   applicationBase + "/saveBusinessProcessDevelopmentArtifact?api-version=" + integrationAPIVersion,
			payload: map[string]any{
				"artifactName": "artifact-a",
				"content":      `{"definition":true}`,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GetArtifact",
			method: http.MethodPost,
			path:   applicationBase + "/getBusinessProcessDevelopmentArtifact?api-version=" + integrationAPIVersion,
			payload: map[string]any{
				"artifactName": "artifact-a",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ValidateArtifact",
			method: http.MethodPost,
			path:   applicationBase + "/validateBusinessProcessDevelopmentArtifact?api-version=" + integrationAPIVersion,
			payload: map[string]any{
				"artifactName": "artifact-a",
				"content":      "valid-content",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListInfrastructureResources",
			method:   http.MethodGet,
			path:     spaceBase + "/infrastructureResources?api-version=" + integrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PatchInfrastructureResource",
			method: http.MethodPatch,
			path:   spaceBase + "/infrastructureResources/default-runtime?api-version=" + integrationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"status": "Updating",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteSpace",
			method:   http.MethodDelete,
			path:     spaceBase + "?api-version=" + integrationAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> %d %s\n", call.name, status, summarizePayload(payload))
	}
}

func summarizePayload(payload map[string]any) string {
	if len(payload) == 0 {
		return "{}"
	}
	if name := strings.TrimSpace(stringValue(payload["name"])); name != "" {
		return "name=" + name
	}
	if values, ok := payload["value"].([]any); ok {
		return fmt.Sprintf("value_count=%d", len(values))
	}
	if valid, ok := payload["isValid"].(bool); ok {
		return fmt.Sprintf("isValid=%t", valid)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("%#v", payload)
	}
	return string(body)
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
