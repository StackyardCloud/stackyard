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

const computeRecommenderAPIVersion = "2025-06-05"

type computeRecommenderClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type computeRecommenderSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *computeRecommenderSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newComputeRecommenderClient(endpoint, account, subscriptionKey string) *computeRecommenderClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"recommenderrp-2025-06-05",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&computeRecommenderSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &computeRecommenderClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *computeRecommenderClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, http.Header, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, nil, 0, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, nil, resp.StatusCode, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.Header, resp.StatusCode, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.Header, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_COMPUTE_RECOMMENDER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMPUTE_RECOMMENDER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")

	client := newComputeRecommenderClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.Compute/locations/" + location + "/placementScores/spot"

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
			path:     "/azure/providers/Microsoft.Compute/operations?api-version=" + computeRecommenderAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetSpotPlacementScoreMetadata",
			method:   http.MethodGet,
			path:     base + "?api-version=" + computeRecommenderAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GenerateSpotPlacementScores",
			method: http.MethodPost,
			path:   base + "/generate?api-version=" + computeRecommenderAPIVersion,
			payload: map[string]any{
				"availabilityZones": []string{"1", "2"},
				"desiredCount":      2,
				"desiredLocations":  []string{"eastus", "westus2"},
				"desiredSizes": []map[string]any{
					{"sku": "Standard_DS2_v2"},
					{"sku": "Standard_D4s_v5"},
				},
			},
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		payload, headers, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, headers, payload)
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

func printJSON(name string, status int, headers http.Header, payload map[string]any) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	if location := headers.Get("Location"); location != "" {
		fmt.Printf("\n== %s (%d) ==\nLocation: %s\n%s\n", name, status, location, string(body))
		return
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
