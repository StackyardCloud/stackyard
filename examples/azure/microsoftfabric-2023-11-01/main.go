package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const microsoftFabricAPIVersion = "2023-11-01"
const microsoftFabricPrefix = "/azure"

type microsoftFabricClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type microsoftFabricSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type microsoftFabricCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *microsoftFabricSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newMicrosoftFabricClient(endpoint, account, subscriptionKey string) *microsoftFabricClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"microsoftfabric-2023-11-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&microsoftFabricSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &microsoftFabricClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *microsoftFabricClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, err)
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

func buildPath(template string, vars map[string]string) string {
	path := template
	for key, value := range vars {
		path = strings.ReplaceAll(path, "{"+key+"}", url.PathEscape(value))
	}
	return microsoftFabricPrefix + path
}

func payloadForMethod(methodPath, location string) any {
	switch methodPath {
	case "fabric-capacities/check-name-availability":
		return map[string]any{
			"name": "fabriccapacity01",
			"type": "Microsoft.Fabric/capacities",
		}
	case "fabric-capacities/create-or-update":
		return map[string]any{
			"location": location,
			"properties": map[string]any{
				"administration": map[string]any{
					"members": []any{"admin1@example.com", "admin2@example.com"},
				},
			},
			"sku": map[string]any{
				"name": "F2",
				"tier": "Fabric",
			},
			"tags": map[string]any{"env": "local"},
		}
	case "fabric-capacities/update":
		return map[string]any{
			"properties": map[string]any{
				"administration": map[string]any{
					"members": []any{"admin2@example.com"},
				},
			},
			"sku":  map[string]any{"name": "F8", "tier": "Fabric"},
			"tags": map[string]any{"env": "local", "tier": "testing"},
		}
	}
	return map[string]any{}
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_MICROSOFTFABRIC_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_MICROSOFTFABRIC_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-microsoftfabric")
	location := getenv("STACKYARD_AZURE_LOCATION", "westcentralus")
	capacityName := getenv("STACKYARD_AZURE_MICROSOFTFABRIC_CAPACITY_NAME", "fabriccapacity01")

	client := newMicrosoftFabricClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":    subscriptionID,
		"resourceGroupName": resourceGroupName,
		"capacityName":      capacityName,
		"location":          location,
	}

	calls := []microsoftFabricCall{
		{
			name:     "fabric_capacities_check_name_availability",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Fabric/locations/{location}/checkNameAvailability?api-version=2023-11-01",
			payload:  payloadForMethod("fabric-capacities/check-name-availability", location),
			statuses: []int{200},
		},
		{
			name:     "fabric_capacities_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Fabric/capacities/{capacityName}?api-version=2023-11-01",
			payload:  payloadForMethod("fabric-capacities/create-or-update", location),
			statuses: []int{200, 201},
		},
		{
			name:     "fabric_capacities_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Fabric/capacities/{capacityName}?api-version=2023-11-01",
			payload:  nil,
			statuses: []int{202, 204},
		},
		{
			name:     "fabric_capacities_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Fabric/capacities/{capacityName}?api-version=2023-11-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "fabric_capacities_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Fabric/capacities?api-version=2023-11-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "fabric_capacities_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Fabric/capacities?api-version=2023-11-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "fabric_capacities_list_skus",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Fabric/skus?api-version=2023-11-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "fabric_capacities_list_skus_for_capacity",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Fabric/capacities/{capacityName}/skus?api-version=2023-11-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "fabric_capacities_resume",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Fabric/capacities/{capacityName}/resume?api-version=2023-11-01",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "fabric_capacities_suspend",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Fabric/capacities/{capacityName}/suspend?api-version=2023-11-01",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "fabric_capacities_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Fabric/capacities/{capacityName}?api-version=2023-11-01",
			payload:  payloadForMethod("fabric-capacities/update", location),
			statuses: []int{200, 202},
		},
		{
			name:     "operations_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Fabric/operations?api-version=2023-11-01",
			payload:  nil,
			statuses: []int{200},
		},
	}

	for _, call := range calls {
		path := buildPath(call.template, vars)
		response, status, err := client.doJSON(ctx, call.method, path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> %d\n", call.name, status)
		if len(response) > 0 {
			encoded, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s marshal response: %v\n", call.name, err)
				os.Exit(1)
			}
			fmt.Println(string(encoded))
		}
	}
}

func ensureStatus(actual int, body []byte, expected ...int) error {
	for _, code := range expected {
		if actual == code {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d, expected one of %v, body: %s", actual, expected, strings.TrimSpace(string(body)))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
