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
	resourcesAPIVersion = "2021-04-01"
	cosmosDBAPIVersion  = "2024-11-15"
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
		"cosmosdb-2024-11-15",
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

func (c *armClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-cosmosdb-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	accountName := getenv("STACKYARD_AZURE_COSMOSDB_ACCOUNT", "stackyard-cosmos")

	fmt.Printf("Stackyard Azure Cosmos DB (cosmosdb-2024-11-15) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	dbAccount := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.DocumentDB/databaseAccounts/" + accountName
	nameCheck := "/azure/providers/Microsoft.DocumentDB/databaseAccountNames/" + accountName + "?api-version=" + cosmosDBAPIVersion
	q := "?api-version=" + cosmosDBAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		// HEAD answers 404 while the account name is free.
		{name: "CheckNameExists", method: http.MethodHead, path: nameCheck, statuses: []int{http.StatusNotFound}},
		{name: "CreateAccount", method: http.MethodPut, path: dbAccount + q, payload: map[string]any{"location": region, "kind": "GlobalDocumentDB", "properties": map[string]any{"databaseAccountOfferType": "Standard", "consistencyPolicy": map[string]any{"defaultConsistencyLevel": "Session"}, "locations": []any{map[string]any{"locationName": region, "failoverPriority": 0}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CheckNameExistsAfterCreate", method: http.MethodHead, path: nameCheck, statuses: []int{http.StatusOK}},
		{name: "ListKeys", method: http.MethodPost, path: dbAccount + "/listKeys" + q, statuses: []int{http.StatusOK}},
		{name: "ListConnectionStrings", method: http.MethodPost, path: dbAccount + "/listConnectionStrings" + q, statuses: []int{http.StatusOK}},
		{name: "CreateSQLDatabase", method: http.MethodPut, path: dbAccount + "/sqlDatabases/orders" + q, payload: map[string]any{"properties": map[string]any{"resource": map[string]any{"id": "orders"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateContainer", method: http.MethodPut, path: dbAccount + "/sqlDatabases/orders/containers/items" + q, payload: map[string]any{"properties": map[string]any{"resource": map[string]any{"id": "items", "partitionKey": map[string]any{"paths": []any{"/customerId"}, "kind": "Hash"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListContainers", method: http.MethodGet, path: dbAccount + "/sqlDatabases/orders/containers" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteAccount", method: http.MethodDelete, path: dbAccount + q, statuses: []int{http.StatusOK}},
		// The databases and containers were removed with the account.
		{name: "GetContainerAfterAccountDelete", method: http.MethodGet, path: dbAccount + "/sqlDatabases/orders/containers/items" + q, statuses: []int{http.StatusNotFound}},
		{name: "DeleteResourceGroup", method: http.MethodDelete, path: groupPath, statuses: []int{http.StatusOK}},
	}

	for _, call := range calls {
		resp, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> HTTP %d\n", call.name, status)
		if len(resp) != 0 {
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
