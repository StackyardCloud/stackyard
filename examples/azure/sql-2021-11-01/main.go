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
	sqlAPIVersion       = "2021-11-01"
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
		"sql-2021-11-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-sql-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	serverName := getenv("STACKYARD_AZURE_SQL_SERVER", "stackyard-sql")
	adminLogin := getenv("STACKYARD_AZURE_SQL_ADMIN_LOGIN", "stackyardadmin")
	adminPassword := getenv("STACKYARD_AZURE_SQL_ADMIN_PASSWORD", "Stackyard-Local-0nly")

	fmt.Printf("Stackyard Azure SQL Database (sql-2021-11-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	server := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Sql/servers/" + serverName
	q := "?api-version=" + sqlAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CheckNameAvailability", method: http.MethodPost, path: base + "/providers/Microsoft.Sql/checkNameAvailability" + q, payload: map[string]any{"name": serverName, "type": "Microsoft.Sql/servers"}, statuses: []int{http.StatusOK}},
		// The password is accepted but never returned.
		{name: "CreateServer", method: http.MethodPut, path: server + q, payload: map[string]any{"location": region, "properties": map[string]any{"administratorLogin": adminLogin, "administratorLoginPassword": adminPassword, "minimalTlsVersion": "1.2"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateElasticPool", method: http.MethodPut, path: server + "/elasticPools/shared" + q, payload: map[string]any{"location": region, "sku": map[string]any{"name": "StandardPool", "tier": "Standard", "capacity": 50}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateDatabase", method: http.MethodPut, path: server + "/databases/orders" + q, payload: map[string]any{"location": region, "sku": map[string]any{"name": "S0", "tier": "Standard"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		// Lists master alongside the new database.
		{name: "ListDatabases", method: http.MethodGet, path: server + "/databases" + q, statuses: []int{http.StatusOK}},
		{name: "AllowAzureServices", method: http.MethodPut, path: server + "/firewallRules/AllowAllWindowsAzureIps" + q, payload: map[string]any{"properties": map[string]any{"startIpAddress": "0.0.0.0", "endIpAddress": "0.0.0.0"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "DeleteServer", method: http.MethodDelete, path: server + q, statuses: []int{http.StatusOK}},
		// The databases were removed with the server.
		{name: "GetDatabaseAfterServerDelete", method: http.MethodGet, path: server + "/databases/orders" + q, statuses: []int{http.StatusNotFound}},
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
