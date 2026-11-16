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
	costManagementAPIVersion = "2023-11-01"
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
		"costmanagement-2023-11-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-cost-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	storageAccount := getenv("STACKYARD_AZURE_COST_STORAGE", "stackyardcost")

	fmt.Printf("Stackyard Azure Cost Management (costmanagement-2023-11-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	cost := base + "/providers/Microsoft.CostManagement"
	groupCost := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.CostManagement"
	storageID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Storage/storageAccounts/" + storageAccount
	viewID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.CostManagement/views/by-service"
	q := "?api-version=" + costManagementAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateExport", method: http.MethodPut, path: cost + "/exports/monthly-actuals" + q, payload: map[string]any{"properties": map[string]any{"format": "Parquet", "definition": map[string]any{"type": "ActualCost", "timeframe": "MonthToDate", "dataSet": map[string]any{"granularity": "Daily"}}, "deliveryInfo": map[string]any{"destination": map[string]any{"resourceId": storageID, "container": "exports", "rootFolderPath": "monthly"}}, "schedule": map[string]any{"status": "Active", "recurrence": "Daily", "recurrencePeriod": map[string]any{"from": "2026-12-01T00:00:00Z", "to": "2027-12-01T00:00:00Z"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "RunExport", method: http.MethodPost, path: cost + "/exports/monthly-actuals/run" + q, statuses: []int{http.StatusOK}},
		{name: "ListExports", method: http.MethodGet, path: cost + "/exports" + q, statuses: []int{http.StatusOK}},
		{name: "CreateView", method: http.MethodPut, path: groupCost + "/views/by-service" + q, payload: map[string]any{"properties": map[string]any{"displayName": "Cost by service", "chart": "StackedColumn", "accumulated": "false", "metric": "ActualCost", "query": map[string]any{"type": "Usage", "timeframe": "MonthToDate", "dataSet": map[string]any{"granularity": "Daily", "grouping": []any{map[string]any{"type": "Dimension", "name": "ServiceName"}}}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateScheduledAction", method: http.MethodPut, path: groupCost + "/scheduledActions/daily-cost" + q, payload: map[string]any{"kind": "Email", "properties": map[string]any{"displayName": "Daily cost", "status": "Enabled", "viewId": viewID, "notification": map[string]any{"to": []any{"ops@example.com"}, "subject": "Daily cost by service"}, "schedule": map[string]any{"frequency": "Daily", "startDate": "2026-12-01T08:00:00Z", "endDate": "2027-12-01T08:00:00Z"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ExecuteScheduledAction", method: http.MethodPost, path: groupCost + "/scheduledActions/daily-cost/execute" + q, statuses: []int{http.StatusOK}},
		// Stackyard bills nothing: queries return their columns and no rows.
		{name: "QueryUsage", method: http.MethodPost, path: cost + "/query" + q, payload: map[string]any{"type": "ActualCost", "timeframe": "MonthToDate", "dataset": map[string]any{"granularity": "Daily", "aggregation": map[string]any{"totalCost": map[string]any{"name": "Cost", "function": "Sum"}}, "grouping": []any{map[string]any{"type": "Dimension", "name": "ResourceGroup"}}}}, statuses: []int{http.StatusOK}},
		{name: "Forecast", method: http.MethodPost, path: groupCost + "/forecast" + q, payload: map[string]any{"type": "Usage", "timeframe": "MonthToDate", "dataset": map[string]any{"granularity": "Daily", "aggregation": map[string]any{"totalCost": map[string]any{"name": "Cost", "function": "Sum"}}}}, statuses: []int{http.StatusOK}},
		{name: "DeleteScheduledAction", method: http.MethodDelete, path: groupCost + "/scheduledActions/daily-cost" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteView", method: http.MethodDelete, path: groupCost + "/views/by-service" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteExport", method: http.MethodDelete, path: cost + "/exports/monthly-actuals" + q, statuses: []int{http.StatusOK}},
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
