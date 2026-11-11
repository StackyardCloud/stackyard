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

const kustoAPIVersion = "2025-02-14"

type kustoClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type kustoSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *kustoSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newKustoClient(endpoint, account, subscriptionKey string) *kustoClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"azurerekusto-2025-02-14",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&kustoSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &kustoClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *kustoClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	if len(expectedStatuses) > 0 {
		ok := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				ok = true
				break
			}
		}
		if !ok {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
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

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_KUSTO_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_KUSTO_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-kusto")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	clusterName := getenv("STACKYARD_AZURE_KUSTO_CLUSTER", "cluster-a")
	databaseName := getenv("STACKYARD_AZURE_KUSTO_DATABASE", "db-a")

	fmt.Printf("Stackyard Azure Kusto example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newKustoClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	clusterBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters/" + clusterName
	databaseBase := clusterBase + "/databases/" + databaseName

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
			path:     "/azure/providers/Microsoft.Kusto/operations?api-version=" + kustoAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSubscriptionSKUs",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.Kusto/skus?api-version=" + kustoAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListRegionalSKUs",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.Kusto/locations/" + region + "/skus?api-version=" + kustoAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateCluster",
			method: http.MethodPut,
			path:   clusterBase + "?api-version=" + kustoAPIVersion,
			payload: map[string]any{
				"location": region,
				"tags":     map[string]any{"env": "dev"},
				"identity": map[string]any{"type": "SystemAssigned"},
				"sku":      map[string]any{"name": "Standard_D13_v2", "tier": "Standard"},
				"zones":    []any{"1", "2"},
				"properties": map[string]any{
					"optimizedAutoscale": map[string]any{"isEnabled": true},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetCluster",
			method:   http.MethodGet,
			path:     clusterBase + "?api-version=" + kustoAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListClustersByResourceGroup",
			method:   http.MethodGet,
			path:     subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters?api-version=" + kustoAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListClustersBySubscription",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.Kusto/clusters?api-version=" + kustoAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListClusterSKUs",
			method:   http.MethodGet,
			path:     clusterBase + "/listSkus?api-version=" + kustoAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "StartCluster",
			method:   http.MethodPost,
			path:     clusterBase + "/start?api-version=" + kustoAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DiagnoseVirtualNetwork",
			method:   http.MethodPost,
			path:     clusterBase + "/diagnoseVirtualNetwork?api-version=" + kustoAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateDatabase",
			method: http.MethodPut,
			path:   databaseBase + "?api-version=" + kustoAPIVersion,
			payload: map[string]any{
				"location": region,
				"kind":     "ReadWrite",
				"properties": map[string]any{
					"softDeletePeriod": "P365D",
					"hotCachePeriod":   "P7D",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetDatabase",
			method:   http.MethodGet,
			path:     databaseBase + "?api-version=" + kustoAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "InviteFollower",
			method:   http.MethodPost,
			path:     databaseBase + "/inviteFollower?api-version=" + kustoAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateDataConnection",
			method: http.MethodPut,
			path:   databaseBase + "/dataConnections/conn-a?api-version=" + kustoAPIVersion,
			payload: map[string]any{
				"kind": "EventHub",
				"properties": map[string]any{
					"consumerGroup": "$Default",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateScript",
			method: http.MethodPut,
			path:   databaseBase + "/scripts/script-a?api-version=" + kustoAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"scriptUrl": "https://example.invalid/script.kql",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateAttachedDatabaseConfiguration",
			method: http.MethodPut,
			path:   clusterBase + "/attachedDatabaseConfigurations/attach-a?api-version=" + kustoAPIVersion,
			payload: map[string]any{
				"location": region,
				"properties": map[string]any{
					"attachedDatabaseNames": []any{"db-a"},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateClusterPrincipalAssignment",
			method: http.MethodPut,
			path:   clusterBase + "/principalAssignments/pa-a?api-version=" + kustoAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"principalId":   "11111111-1111-1111-1111-111111111111",
					"principalType": "App",
					"role":          "AllDatabasesAdmin",
					"tenantId":      "22222222-2222-2222-2222-222222222222",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateDatabasePrincipalAssignment",
			method: http.MethodPut,
			path:   databaseBase + "/principalAssignments/db-pa-a?api-version=" + kustoAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"principalId":   "11111111-1111-1111-1111-111111111111",
					"principalType": "App",
					"role":          "Admin",
					"tenantId":      "22222222-2222-2222-2222-222222222222",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateManagedPrivateEndpoint",
			method: http.MethodPut,
			path:   clusterBase + "/managedPrivateEndpoints/mpe-a?api-version=" + kustoAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"groupId":               "cluster",
					"privateLinkResourceId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListPrivateLinkResources",
			method:   http.MethodGet,
			path:     clusterBase + "/privateLinkResources?api-version=" + kustoAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetPrivateLinkResource",
			method:   http.MethodGet,
			path:     clusterBase + "/privateLinkResources/cluster?api-version=" + kustoAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteCluster",
			method:   http.MethodDelete,
			path:     clusterBase + "?api-version=" + kustoAPIVersion,
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
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n%s\n", name, status, string(formatted))
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
