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

const aiFoundryAccountManagementAPIVersion = "2025-06-01"

type aiFoundryAccountManagementClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type sharedKeyAndSubscriptionPolicy struct {
	account         string
	subscriptionKey string
}

func (p *sharedKeyAndSubscriptionPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newAIFoundryAccountManagementClient(endpoint, account, subscriptionKey string) *aiFoundryAccountManagementClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"ai-foundry-account-management-2025-06-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &aiFoundryAccountManagementClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *aiFoundryAccountManagementClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
		return nil, resp.StatusCode, fmt.Errorf("read response body %s %s: %w", method, path, readErr)
	}

	if len(expectedStatuses) == 0 {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else {
		matched := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				matched = true
				break
			}
		}
		if !matched {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	}

	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-foundry")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	accountName := getenv("STACKYARD_AZURE_AI_FOUNDRY_ACCOUNT", "foundry-a")
	projectName := getenv("STACKYARD_AZURE_AI_FOUNDRY_PROJECT", "project-a")
	connectionName := getenv("STACKYARD_AZURE_AI_FOUNDRY_CONNECTION", "conn-a")
	vectorStoreConnectionName := getenv("STACKYARD_AZURE_AI_FOUNDRY_VECTOR_STORE_CONNECTION", "vsconn-a")

	fmt.Printf("Stackyard Azure AI Foundry Account Management (account-management-2025-06-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAIFoundryAccountManagementClient(endpoint, account, subscriptionKey)
	scope := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.CognitiveServices/accounts/" + accountName

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
			path:     "/azure/providers/Microsoft.CognitiveServices/operations?api-version=" + aiFoundryAccountManagementAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateAccount",
			method: http.MethodPut,
			path:   scope + "?api-version=" + aiFoundryAccountManagementAPIVersion,
			payload: map[string]any{
				"location": region,
				"kind":     "AIServices",
				"sku":      map[string]any{"name": "S0"},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusNotImplemented},
		},
		{
			name:     "CheckDomainAvailability",
			method:   http.MethodPost,
			path:     "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.CognitiveServices/checkDomainAvailability?api-version=" + aiFoundryAccountManagementAPIVersion,
			payload:  map[string]any{"subdomainName": "stackyard-foundry", "type": "Microsoft.CognitiveServices/accounts"},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "CheckSkuAvailability",
			method:   http.MethodPost,
			path:     "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.CognitiveServices/checkSkuAvailability?api-version=" + aiFoundryAccountManagementAPIVersion,
			payload:  map[string]any{"kind": "AIServices", "skus": []string{"S0"}, "type": "Microsoft.CognitiveServices/accounts"},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "CreateProject",
			method:   http.MethodPut,
			path:     scope + "/projects/" + projectName + "?api-version=" + aiFoundryAccountManagementAPIVersion,
			payload:  map[string]any{"properties": map[string]any{"description": "stackyard project"}},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusNotImplemented},
		},
		{
			name:     "ListProjects",
			method:   http.MethodGet,
			path:     scope + "/projects?api-version=" + aiFoundryAccountManagementAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "CreateProjectConnection",
			method:   http.MethodPut,
			path:     scope + "/projects/" + projectName + "/connections/" + connectionName + "?api-version=" + aiFoundryAccountManagementAPIVersion,
			payload:  map[string]any{"properties": map[string]any{"category": "AzureResource", "target": "https://example"}},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusNotImplemented},
		},
		{
			name:     "ListProjectConnections",
			method:   http.MethodGet,
			path:     scope + "/projects/" + projectName + "/connections?api-version=" + aiFoundryAccountManagementAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListAccountCapabilityHosts",
			method:   http.MethodGet,
			path:     scope + "/accountCapabilityHosts?api-version=" + aiFoundryAccountManagementAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListProjectCapabilityHosts",
			method:   http.MethodGet,
			path:     scope + "/projects/" + projectName + "/projectCapabilityHosts?api-version=" + aiFoundryAccountManagementAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "CreateVectorStoreConnection",
			method:   http.MethodPut,
			path:     scope + "/vectorStoreConnections/" + vectorStoreConnectionName + "?api-version=" + aiFoundryAccountManagementAPIVersion,
			payload:  map[string]any{"properties": map[string]any{"type": "azureBlob", "target": "https://example"}},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusNotImplemented},
		},
		{
			name:     "ListVectorStoreConnections",
			method:   http.MethodGet,
			path:     scope + "/vectorStoreConnections?api-version=" + aiFoundryAccountManagementAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListUsages",
			method:   http.MethodGet,
			path:     "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.CognitiveServices/locations/" + region + "/usages?api-version=" + aiFoundryAccountManagementAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		_, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		if status == http.StatusNotImplemented {
			notImplementedCount++
			fmt.Printf("Route is recognized but not implemented yet: %s\n", call.path)
			continue
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All ai-foundry account-management routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
