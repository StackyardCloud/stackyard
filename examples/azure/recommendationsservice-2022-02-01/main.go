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

const (
	resourcesAPIVersion       = "2021-04-01"
	recommendationsAPIVersion = "2022-02-01"
)

type armClient struct {
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

func newARMClient(endpoint, account, subscriptionKey string) *armClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"recommendationsservice-2022-02-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &armClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *armClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (any, int, error) {
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
		return nil, resp.StatusCode, nil
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-recommendations-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	accountName := getenv("STACKYARD_AZURE_RECOMMENDATIONS_ACCOUNT", "stackyard-reco")

	fmt.Printf("Stackyard Azure Intelligent Recommendations (recommendationsservice-2022-02-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	acct := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.RecommendationsService/accounts/" + accountName
	q := "?api-version=" + recommendationsAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CheckNameAvailability", method: http.MethodPost, path: base + "/providers/Microsoft.RecommendationsService/checkNameAvailability" + q, payload: map[string]any{"type": "Microsoft.RecommendationsService/accounts", "name": accountName}, statuses: []int{http.StatusOK}},
		{name: "CreateAccount", method: http.MethodPut, path: acct + q, payload: map[string]any{"location": region, "tags": map[string]any{"Environment": "Prod"}, "properties": map[string]any{"configuration": "Capacity", "endpointAuthentications": []any{map[string]any{"aadTenantID": "tenant", "principalID": "oid", "principalType": "User"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "UpdateAccount", method: http.MethodPatch, path: acct + q, payload: map[string]any{"properties": map[string]any{"cors": []any{map[string]any{"allowedOrigins": []any{"http://siteA.com"}, "allowedMethods": []any{"GET", "POST"}}}}}, statuses: []int{http.StatusOK}},
		{name: "CreateModeling", method: http.MethodPut, path: acct + "/modeling/c1" + q, payload: map[string]any{"location": region, "properties": map[string]any{"features": "Standard", "frequency": "High", "size": "Medium", "inputData": map[string]any{"connectionString": "https://storageAccount.blob.core.windows.net/container/root"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "UpdateModeling", method: http.MethodPatch, path: acct + "/modeling/c1" + q, payload: map[string]any{"properties": map[string]any{"frequency": "Low"}}, statuses: []int{http.StatusOK}},
		{name: "GetAccountStatus", method: http.MethodPost, path: acct + "/status" + q, statuses: []int{http.StatusOK}},
		{name: "CreateServiceEndpoint", method: http.MethodPut, path: acct + "/serviceEndpoints/s1" + q, payload: map[string]any{"location": region, "properties": map[string]any{"preAllocatedCapacity": 100}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "UpdateServiceEndpoint", method: http.MethodPatch, path: acct + "/serviceEndpoints/s1" + q, payload: map[string]any{"tags": map[string]any{"Environment": "Test"}}, statuses: []int{http.StatusOK}},
		{name: "ListServiceEndpoints", method: http.MethodGet, path: acct + "/serviceEndpoints" + q, statuses: []int{http.StatusOK}},
		{name: "ListModeling", method: http.MethodGet, path: acct + "/modeling" + q, statuses: []int{http.StatusOK}},
		{name: "ListAccounts", method: http.MethodGet, path: base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.RecommendationsService/accounts" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteServiceEndpoint", method: http.MethodDelete, path: acct + "/serviceEndpoints/s1" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteModeling", method: http.MethodDelete, path: acct + "/modeling/c1" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteAccount", method: http.MethodDelete, path: acct + q, statuses: []int{http.StatusOK}},
		{name: "DeleteResourceGroup", method: http.MethodDelete, path: groupPath, statuses: []int{http.StatusOK}},
	}

	for _, call := range calls {
		resp, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> HTTP %d\n", call.name, status)
		if resp != nil {
			encoded, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(encoded))
		}
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
