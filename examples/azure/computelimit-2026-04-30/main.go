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

const computeLimitAPIVersion = "2026-04-30"

type computeLimitClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type computeLimitSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *computeLimitSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newComputeLimitClient(endpoint, account, subscriptionKey string) *computeLimitClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"computelimit-2026-04-30",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&computeLimitSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &computeLimitClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *computeLimitClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_COMPUTE_LIMIT_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMPUTE_LIMIT_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	guestSubscriptionID := getenv("STACKYARD_AZURE_GUEST_SUBSCRIPTION_ID", "11111111-1111-1111-1111-111111111111")

	client := newComputeLimitClient(endpoint, account, subscriptionKey)
	locationBase := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.ComputeLimit/locations/" + location

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
			path:     "/azure/providers/Microsoft.ComputeLimit/operations?api-version=" + computeLimitAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFeatures",
			method:   http.MethodGet,
			path:     locationBase + "/features?api-version=" + computeLimitAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "EnableCrossSubscriptionSharing",
			method:   http.MethodPost,
			path:     locationBase + "/features/CrossSubscriptionSharing/enable?api-version=" + computeLimitAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateSharedLimit",
			method: http.MethodPut,
			path:   locationBase + "/sharedLimits/dsv5-share?api-version=" + computeLimitAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"resourceName": map[string]any{
						"value":          "standardDSv5Family",
						"localizedValue": "Standard DSv5 Family",
					},
					"limit": 64,
					"unit":  "Count",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateGuestSubscription",
			method: http.MethodPut,
			path:   locationBase + "/guestSubscriptions/" + guestSubscriptionID + "?api-version=" + computeLimitAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": "Guest Subscription A",
					"notes":       "linked for eastus",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListVMFamilies",
			method:   http.MethodGet,
			path:     locationBase + "/vmFamilies?$filter=generalpurpose&api-version=" + computeLimitAPIVersion,
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
