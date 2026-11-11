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

const batchManagementAPIVersion = "2025-06-01"

type batchManagementClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type batchManagementSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *batchManagementSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newBatchManagementClient(endpoint, account, subscriptionKey string) *batchManagementClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"batchmanagement-2025-06-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&batchManagementSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &batchManagementClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *batchManagementClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_BATCH_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_BATCH_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-batch")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	batchAccountName := getenv("STACKYARD_AZURE_BATCH_ACCOUNT_NAME", "batchacct123")

	fmt.Printf("Stackyard Azure Batch Management example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newBatchManagementClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	accountBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Batch/batchAccounts/" + batchAccountName

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
			path:     "/azure/providers/Microsoft.Batch/operations?api-version=" + batchManagementAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path:   subscriptionBase + "/providers/Microsoft.Batch/locations/" + location + "/checkNameAvailability?api-version=" + batchManagementAPIVersion,
			payload: map[string]any{
				"name": batchAccountName,
				"type": "Microsoft.Batch/batchAccounts",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListQuotas",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.Batch/locations/" + location + "/quotas?api-version=" + batchManagementAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSupportedVirtualMachineSkus",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.Batch/locations/" + location + "/supportedVirtualMachineSkus?api-version=" + batchManagementAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateBatchAccount",
			method: http.MethodPut,
			path:   accountBase + "?api-version=" + batchManagementAPIVersion,
			payload: map[string]any{
				"location": location,
				"identity": map[string]any{"type": "SystemAssigned"},
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"allowedAuthenticationModes": []any{"SharedKey", "AAD"},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetBatchAccount",
			method:   http.MethodGet,
			path:     accountBase + "?api-version=" + batchManagementAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListKeys",
			method:   http.MethodPost,
			path:     accountBase + "/listKeys?api-version=" + batchManagementAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateApplication",
			method: http.MethodPut,
			path:   accountBase + "/applications/appa?api-version=" + batchManagementAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"allowUpdates": true},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "CreateApplicationPackage",
			method:   http.MethodPut,
			path:     accountBase + "/applications/appa/versions/v1?api-version=" + batchManagementAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ActivateApplicationPackage",
			method:   http.MethodPost,
			path:     accountBase + "/applications/appa/versions/v1/activate?api-version=" + batchManagementAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreatePool",
			method: http.MethodPut,
			path:   accountBase + "/pools/poola?api-version=" + batchManagementAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"vmSize": "Standard_DS2_v2",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "PatchPrivateEndpointConnection",
			method: http.MethodPatch,
			path:   accountBase + "/privateEndpointConnections/peca?api-version=" + batchManagementAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"privateEndpoint": map[string]any{
						"id": "/subscriptions/" + subscriptionID + "/resourceGroups/rg-network/providers/Microsoft.Network/privateEndpoints/pe-a",
					},
					"groupIds": []any{"batchAccount"},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListPrivateLinkResources",
			method:   http.MethodGet,
			path:     accountBase + "/privateLinkResources?api-version=" + batchManagementAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ReconcileNSPConfiguration",
			method:   http.MethodPost,
			path:     accountBase + "/networkSecurityPerimeterConfigurations/default/reconcile?api-version=" + batchManagementAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
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
		fatalf("%s: marshal response: %v", name, err)
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
