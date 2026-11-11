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

const consumptionAPIVersion = "2024-08-01"

type consumptionClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type consumptionSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *consumptionSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newConsumptionClient(endpoint, account, subscriptionKey string) *consumptionClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"consumption-2024-08-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&consumptionSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &consumptionClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *consumptionClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_CONSUMPTION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CONSUMPTION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")

	client := newConsumptionClient(endpoint, account, subscriptionKey)
	subscriptionScope := "/azure/subscriptions/" + subscriptionID
	budgetName := "monthly-prod"

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
			path:     "/azure/providers/Microsoft.Consumption/operations?api-version=" + consumptionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateBudget",
			method: http.MethodPut,
			path:   subscriptionScope + "/providers/Microsoft.Consumption/budgets/" + budgetName + "?api-version=" + consumptionAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"amount":     1500,
					"category":   "Cost",
					"timeGrain":  "Monthly",
					"timePeriod": map[string]any{"startDate": "2026-04-01", "endDate": "2026-04-30"},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListBudgets",
			method:   http.MethodGet,
			path:     subscriptionScope + "/providers/Microsoft.Consumption/budgets?api-version=" + consumptionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListCharges",
			method:   http.MethodGet,
			path:     subscriptionScope + "/providers/Microsoft.Consumption/charges?api-version=" + consumptionAPIVersion + "&top=2",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetTagSummary",
			method:   http.MethodGet,
			path:     subscriptionScope + "/providers/Microsoft.Consumption/tags?api-version=" + consumptionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetBillingBalance",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.Billing/billingAccounts/mca-account/providers/Microsoft.Consumption/balances?api-version=" + consumptionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetAggregatedCost",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.Management/managementGroups/contoso-platform/providers/Microsoft.Consumption/aggregatedcost?api-version=" + consumptionAPIVersion,
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

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
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
