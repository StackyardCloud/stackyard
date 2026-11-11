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

const durableTaskAPIVersion = "2026-02-01"
const durableTaskPrefix = "/azure"

type durableTaskClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type durableTaskSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type durableTaskCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *durableTaskSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDurableTaskClient(endpoint, account, subscriptionKey string) *durableTaskClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"durabletask-2026-02-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&durableTaskSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &durableTaskClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *durableTaskClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return durableTaskPrefix + path
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DURABLETASK_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DURABLETASK_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-durabletask")
	location := getenv("STACKYARD_AZURE_LOCATION", "northcentralus")
	schedulerName := getenv("STACKYARD_AZURE_DURABLETASK_SCHEDULER_NAME", "testscheduler")
	taskHubName := getenv("STACKYARD_AZURE_DURABLETASK_TASK_HUB_NAME", "testtaskhub")
	privateEndpointConnectionName := getenv("STACKYARD_AZURE_DURABLETASK_PRIVATE_ENDPOINT_CONNECTION_NAME", "peconnection01")
	privateLinkResourceName := getenv("STACKYARD_AZURE_DURABLETASK_PRIVATE_LINK_RESOURCE_NAME", "default")

	client := newDurableTaskClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":                subscriptionID,
		"resourceGroupName":             resourceGroupName,
		"schedulerName":                 schedulerName,
		"taskHubName":                   taskHubName,
		"privateEndpointConnectionName": privateEndpointConnectionName,
		"privateLinkResourceName":       privateLinkResourceName,
	}

	payloadSchedulerCreate := map[string]any{
		"location": location,
		"properties": map[string]any{
			"ipAllowlist": []any{"10.0.0.0/8"},
			"sku": map[string]any{
				"name": "Dedicated",
			},
		},
		"tags": map[string]any{
			"department": "research",
			"env":        "local",
		},
	}

	payloadSchedulerUpdate := map[string]any{
		"tags": map[string]any{
			"env": "local-updated",
		},
		"properties": map[string]any{
			"ipAllowlist": []any{"10.0.0.0/8"},
			"sku": map[string]any{
				"name":     "Dedicated",
				"capacity": 3,
			},
		},
	}

	payloadPrivateEndpointConnectionCreate := map[string]any{
		"properties": map[string]any{
			"privateEndpoint": map[string]any{},
			"privateLinkServiceConnectionState": map[string]any{
				"status":          "Pending",
				"actionsRequired": "None",
				"description":     "awaiting approval",
			},
		},
	}

	payloadPrivateEndpointConnectionUpdate := map[string]any{
		"properties": map[string]any{
			"privateLinkServiceConnectionState": map[string]any{
				"status": "Approved",
			},
		},
	}

	payloadTaskHubCreate := map[string]any{
		"properties": map[string]any{},
	}

	payloadRetentionPolicyCreate := map[string]any{
		"properties": map[string]any{
			"retentionPolicies": []any{
				map[string]any{"retentionPeriodInDays": 30},
				map[string]any{"retentionPeriodInDays": 10, "orchestrationState": "Failed"},
			},
		},
	}

	payloadRetentionPolicyUpdate := map[string]any{
		"properties": map[string]any{
			"retentionPolicies": []any{
				map[string]any{"retentionPeriodInDays": 30},
				map[string]any{"retentionPeriodInDays": 10, "orchestrationState": "Failed"},
				map[string]any{"retentionPeriodInDays": 24, "orchestrationState": "Completed"},
			},
		},
	}

	calls := []durableTaskCall{
		{
			name:     "operations_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.DurableTask/operations?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "schedulers_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}?api-version=2026-02-01",
			payload:  payloadSchedulerCreate,
			statuses: []int{200, 201},
		},
		{
			name:     "schedulers_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "schedulers_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.DurableTask/schedulers?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "schedulers_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "schedulers_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}?api-version=2026-02-01",
			payload:  payloadSchedulerUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "schedulers_create_or_update_private_endpoint_connection",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-02-01",
			payload:  payloadPrivateEndpointConnectionCreate,
			statuses: []int{200, 201},
		},
		{
			name:     "schedulers_get_private_endpoint_connection",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "schedulers_list_private_endpoint_connections",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateEndpointConnections?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "schedulers_update_private_endpoint_connection",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-02-01",
			payload:  payloadPrivateEndpointConnectionUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "schedulers_list_private_links",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateLinkResources?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "schedulers_get_private_link",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateLinkResources/{privateLinkResourceName}?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "task_hubs_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/taskHubs/{taskHubName}?api-version=2026-02-01",
			payload:  payloadTaskHubCreate,
			statuses: []int{200, 201},
		},
		{
			name:     "task_hubs_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/taskHubs/{taskHubName}?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "task_hubs_list_by_scheduler",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/taskHubs?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "retention_policies_create_or_replace",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/retentionPolicies/default?api-version=2026-02-01",
			payload:  payloadRetentionPolicyCreate,
			statuses: []int{200, 201},
		},
		{
			name:     "retention_policies_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/retentionPolicies/default?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "retention_policies_list_by_scheduler",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/retentionPolicies?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "retention_policies_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/retentionPolicies/default?api-version=2026-02-01",
			payload:  payloadRetentionPolicyUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "task_hubs_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/taskHubs/{taskHubName}?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{202, 204},
		},
		{
			name:     "schedulers_delete_private_endpoint_connection",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{202, 204},
		},
		{
			name:     "retention_policies_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/retentionPolicies/default?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{202, 204},
		},
		{
			name:     "schedulers_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}?api-version=2026-02-01",
			payload:  nil,
			statuses: []int{202, 204},
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
