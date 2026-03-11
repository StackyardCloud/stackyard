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

const accountManagementAPIVersion = "2024-10-01"

type accountManagementClient struct {
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

func newAccountManagementClient(endpoint, account, subscriptionKey string) *accountManagementClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"account-management-2024-10-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &accountManagementClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *accountManagementClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

	if payload != nil {
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-ai")
	accountName := getenv("STACKYARD_AZURE_ACCOUNT_NAME", "stackyard-ai")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	deploymentName := getenv("STACKYARD_AZURE_DEPLOYMENT_NAME", "gpt4o")

	fmt.Printf("Stackyard Azure AI Services Account Management (account-management-2024-10-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAccountManagementClient(endpoint, account, subscriptionKey)

	accountBase := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.CognitiveServices/accounts/" + accountName
	subscriptionBase := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.CognitiveServices"

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "CreateOrUpdateAccount",
			method: http.MethodPut,
			path:   accountBase + "?api-version=" + accountManagementAPIVersion,
			payload: map[string]any{
				"location": location,
				"kind":     "OpenAI",
				"sku": map[string]any{
					"name": "S0",
				},
				"properties": map[string]any{
					"customSubDomainName": "stackyard-ai",
				},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetAccount",
			method:   http.MethodGet,
			path:     accountBase + "?api-version=" + accountManagementAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListAccounts",
			method:   http.MethodGet,
			path:     "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.CognitiveServices/accounts?api-version=" + accountManagementAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CheckDomainAvailability",
			method: http.MethodPost,
			path:   subscriptionBase + "/checkDomainAvailability?api-version=" + accountManagementAPIVersion,
			payload: map[string]any{
				"subdomainName": "stackyard-ai",
				"type":          "Microsoft.CognitiveServices/accounts",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "GetAccountDeployment",
			method: http.MethodGet,
			path:   accountBase + "/deployments/" + deploymentName + "?api-version=" + accountManagementAPIVersion,
			statuses: []int{
				http.StatusOK,
				http.StatusNotImplemented,
			},
		},
		{
			name:   "ListResourceSkus",
			method: http.MethodGet,
			path:   subscriptionBase + "/skus?api-version=" + accountManagementAPIVersion,
			statuses: []int{
				http.StatusOK,
				http.StatusNotImplemented,
			},
		},
		{
			name:   "ListProviderOperations",
			method: http.MethodGet,
			path:   "/azure/providers/Microsoft.CognitiveServices/operations?api-version=" + accountManagementAPIVersion,
			statuses: []int{
				http.StatusOK,
				http.StatusNotImplemented,
			},
		},
		{
			name:   "ListUsages",
			method: http.MethodGet,
			path:   subscriptionBase + "/locations/" + location + "/usages?api-version=" + accountManagementAPIVersion,
			statuses: []int{
				http.StatusOK,
				http.StatusNotImplemented,
			},
		},
		{
			name:   "CalculateModelCapacity",
			method: http.MethodPost,
			path:   subscriptionBase + "/locations/" + location + ":calculateModelCapacity?api-version=" + accountManagementAPIVersion,
			payload: map[string]any{
				"model": map[string]any{
					"name": "gpt-4o-mini",
				},
				"sku": map[string]any{
					"name": "S0",
				},
			},
			statuses: []int{
				http.StatusOK,
				http.StatusNotImplemented,
			},
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
		fmt.Println("All account-management routes are staged in this Stackyard build.")
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
