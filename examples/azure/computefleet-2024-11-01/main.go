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

const computeFleetAPIVersion = "2024-11-01"

type computeFleetClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type computeFleetSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *computeFleetSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newComputeFleetClient(endpoint, account, subscriptionKey string) *computeFleetClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"computefleet-2024-11-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&computeFleetSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &computeFleetClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *computeFleetClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, readErr)
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
	account := getenv("STACKYARD_AZURE_COMPUTE_FLEET_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMPUTE_FLEET_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-compute-fleet")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	fleetName := getenv("STACKYARD_AZURE_COMPUTE_FLEET_NAME", "fleet-a")

	fmt.Printf("Stackyard Azure Compute Fleet example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newComputeFleetClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	resourceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureFleet/fleets/" + fleetName

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
			path:     "/azure/providers/Microsoft.AzureFleet/operations?api-version=" + computeFleetAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateFleet",
			method: http.MethodPut,
			path:   resourceBase + "?api-version=" + computeFleetAPIVersion,
			payload: map[string]any{
				"location": region,
				"tags":     map[string]any{"env": "dev"},
				"zones":    []any{"1"},
				"properties": map[string]any{
					"computeProfile": map[string]any{
						"platformFaultDomainCount": 1,
					},
					"vmSizesProfile": map[string]any{
						"vmSizes": []any{
							map[string]any{"name": "Standard_D2s_v5", "rank": 1},
						},
					},
					"regularPriorityProfile": map[string]any{
						"minCount": 1,
						"capacity": 2,
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetFleet",
			method:   http.MethodGet,
			path:     resourceBase + "?api-version=" + computeFleetAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFleetsByResourceGroup",
			method:   http.MethodGet,
			path:     subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureFleet/fleets?api-version=" + computeFleetAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFleetsBySubscription",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.AzureFleet/fleets?api-version=" + computeFleetAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListVirtualMachineScaleSets",
			method:   http.MethodGet,
			path:     resourceBase + "/virtualMachineScaleSets?api-version=" + computeFleetAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PatchFleet",
			method: http.MethodPatch,
			path:   resourceBase + "?api-version=" + computeFleetAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"env":   "stage",
					"owner": "stackyard",
				},
				"properties": map[string]any{
					"spotPriorityProfile": map[string]any{
						"minCount": 0,
						"capacity": 1,
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteFleet",
			method:   http.MethodDelete,
			path:     resourceBase + "?api-version=" + computeFleetAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
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
		fatalf("%s: marshal response: %v", name, err)
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
