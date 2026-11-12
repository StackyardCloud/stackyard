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

const carbonAPIVersion = "2025-04-01"

type carbonClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type carbonSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *carbonSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCarbonClient(endpoint, account, subscriptionKey string) *carbonClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"carbon-2025-04-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&carbonSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &carbonClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *carbonClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_CARBON_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CARBON_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")

	fmt.Printf("Stackyard Azure Carbon example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCarbonClient(endpoint, account, subscriptionKey)
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
			path:     "/azure/providers/Microsoft.Carbon/operations?api-version=" + carbonAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "QueryAvailableDateRange",
			method:   http.MethodPost,
			path:     "/azure/providers/Microsoft.Carbon/queryCarbonEmissionDataAvailableDateRange?api-version=" + carbonAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "QueryOverallSummaryReport",
			method: http.MethodPost,
			path:   "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version=" + carbonAPIVersion,
			payload: map[string]any{
				"reportType":       "Overall Summary Report",
				"subscriptionList": []any{subscriptionID, "11111111-1111-1111-1111-111111111111"},
				"carbonScopeList":  []any{"Scope1", "Scope2", "Scope3"},
				"dateRange": map[string]any{
					"startDate": "2025-01-01",
					"endDate":   "2025-02-01",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "QueryItemDetailsReport",
			method: http.MethodPost,
			path:   "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version=" + carbonAPIVersion,
			payload: map[string]any{
				"reportType":       "Item Details Report",
				"subscriptionList": []any{subscriptionID},
				"carbonScopeList":  []any{"Scope1", "Scope2"},
				"dateRange": map[string]any{
					"startDate": "2025-01-01",
					"endDate":   "2025-01-31",
				},
				"pageSize": 1,
			},
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
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("format %s response: %v", name, err)
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
