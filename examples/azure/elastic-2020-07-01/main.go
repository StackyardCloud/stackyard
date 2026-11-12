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

const elasticAPIVersion = "2020-07-01"
const elasticPrefix = "/azure"

type elasticClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type elasticSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type elasticCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *elasticSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newElasticClient(endpoint, account, subscriptionKey string) *elasticClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"elastic-2020-07-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&elasticSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &elasticClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *elasticClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return elasticPrefix + path
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_ELASTIC_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_ELASTIC_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-elastic")
	location := getenv("STACKYARD_AZURE_LOCATION", "westus2")
	monitorName := getenv("STACKYARD_AZURE_ELASTIC_MONITOR_NAME", "myMonitor")
	ruleSetName := getenv("STACKYARD_AZURE_ELASTIC_RULE_SET_NAME", "default")
	vmName := getenv("STACKYARD_AZURE_ELASTIC_VM_NAME", "elastic-vm-01")

	client := newElasticClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":    subscriptionID,
		"resourceGroupName": resourceGroupName,
		"monitorName":       monitorName,
		"ruleSetName":       ruleSetName,
	}

	vmResourceID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s",
		subscriptionID,
		resourceGroupName,
		vmName,
	)

	monitorCreatePayload := map[string]any{
		"location": location,
		"sku": map[string]any{
			"name": "free_Monthly",
		},
		"tags": map[string]any{
			"environment": "local",
			"service":     "elastic",
		},
		"properties": map[string]any{
			"monitoringStatus": "Enabled",
		},
	}

	monitorUpdatePayload := map[string]any{
		"tags": map[string]any{
			"environment": "local-updated",
			"service":     "elastic",
		},
	}

	tagRulesPayload := map[string]any{
		"properties": map[string]any{
			"logRules": map[string]any{
				"sendAadLogs":          false,
				"sendSubscriptionLogs": true,
				"sendActivityLogs":     true,
				"filteringTags": []any{
					map[string]any{
						"name":   "Environment",
						"value":  "Prod",
						"action": "Include",
					},
					map[string]any{
						"name":   "Environment",
						"value":  "Dev",
						"action": "Exclude",
					},
				},
			},
		},
	}

	vmCollectionUpdatePayload := map[string]any{
		"operationName": "Add",
		"vmResourceId":  vmResourceID,
	}

	calls := []elasticCall{
		{
			name:     "operations_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Elastic/operations?api-version=2020-07-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_create",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}?api-version=2020-07-01",
			payload:  monitorCreatePayload,
			statuses: []int{200, 201},
		},
		{
			name:     "monitors_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}?api-version=2020-07-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors?api-version=2020-07-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Elastic/monitors?api-version=2020-07-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}?api-version=2020-07-01",
			payload:  monitorUpdatePayload,
			statuses: []int{200},
		},
		{
			name:     "deployment_info_list",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/listDeploymentInfo?api-version=2020-07-01",
			statuses: []int{200},
		},
		{
			name:     "monitored_resources_list",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/listMonitoredResources?api-version=2020-07-01",
			statuses: []int{200},
		},
		{
			name:     "tag_rules_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2020-07-01",
			payload:  tagRulesPayload,
			statuses: []int{200},
		},
		{
			name:     "tag_rules_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2020-07-01",
			statuses: []int{200},
		},
		{
			name:     "tag_rules_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/tagRules?api-version=2020-07-01",
			statuses: []int{200},
		},
		{
			name:     "vm_collection_update",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/vmCollectionUpdate?api-version=2020-07-01",
			payload:  vmCollectionUpdatePayload,
			statuses: []int{200},
		},
		{
			name:     "vm_host_list",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/listVMHost?api-version=2020-07-01",
			statuses: []int{200},
		},
		{
			name:     "vm_ingestion_details",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/vmIngestionDetails?api-version=2020-07-01",
			statuses: []int{200},
		},
	}

	for _, call := range calls {
		path := buildPath(call.template, vars)
		response, status, err := client.doJSON(ctx, call.method, path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, response)
	}

	cleanup := []elasticCall{
		{
			name:     "tag_rules_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2020-07-01",
			statuses: []int{200, 202},
		},
		{
			name:     "monitors_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}?api-version=2020-07-01",
			statuses: []int{200, 202},
		},
	}

	for _, call := range cleanup {
		path := buildPath(call.template, vars)
		response, status, err := client.doJSON(ctx, call.method, path, nil, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, response)
	}
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func printJSON(name string, status int, payload map[string]any) {
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("%s -> %d\n%s\n", name, status, string(formatted))
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
