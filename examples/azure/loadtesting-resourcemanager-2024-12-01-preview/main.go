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

const loadTestingAPIVersion = "2024-12-01-preview"

type loadTestingClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type loadTestingSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *loadTestingSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newLoadTestingClient(endpoint, account, subscriptionKey string) *loadTestingClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"loadtesting-resourcemanager-2024-12-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&loadTestingSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &loadTestingClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *loadTestingClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_LOADTESTING_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_LOADTESTING_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-loadtesting")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	loadTestName := getenv("STACKYARD_AZURE_LOADTEST_NAME", "loadtest-a")
	parentResourceURI := getenv("STACKYARD_AZURE_LOADTEST_PARENT_RESOURCE_URI", "/subscriptions/"+subscriptionID+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Web/sites/site-a")

	fmt.Printf("Stackyard Azure Load Testing example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newLoadTestingClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	resourceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/loadTests/" + loadTestName
	parentBase := "/azure" + parentResourceURI + "/providers/Microsoft.LoadTestService"
	loadTestID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/loadTests/" + loadTestName

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
			path:     "/azure/providers/Microsoft.LoadTestService/operations?api-version=" + loadTestingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateLoadTest",
			method: http.MethodPut,
			path:   resourceBase + "?api-version=" + loadTestingAPIVersion,
			payload: map[string]any{
				"location": region,
				"identity": map[string]any{"type": "SystemAssigned"},
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"description": "stackyard load test",
					"encryption":  map[string]any{"type": "SystemAssigned"},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetLoadTest",
			method:   http.MethodGet,
			path:     resourceBase + "?api-version=" + loadTestingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListByResourceGroup",
			method:   http.MethodGet,
			path:     subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/loadTests?api-version=" + loadTestingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListBySubscription",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.LoadTestService/loadTests?api-version=" + loadTestingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListOutboundNetworkDependenciesEndpoints",
			method:   http.MethodGet,
			path:     resourceBase + "/outboundNetworkDependenciesEndpoints?api-version=" + loadTestingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetMaxMonthlyVirtualUserHours",
			method:   http.MethodGet,
			path:     resourceBase + "/limits/maxMonthlyVirtualUserHours?api-version=" + loadTestingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "SetMaxMonthlyVirtualUserHours",
			method: http.MethodPost,
			path:   resourceBase + "/limits/maxMonthlyVirtualUserHours/set?api-version=" + loadTestingAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"value": 18000},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateLoadTestMapping",
			method: http.MethodPut,
			path:   parentBase + "/loadTestMappings/mapping-a?api-version=" + loadTestingAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"sourceResourceId":           parentResourceURI,
					"azureLoadTestingResourceId": loadTestID,
					"testId":                     "test-a",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListLoadTestMappings",
			method:   http.MethodGet,
			path:     parentBase + "/loadTestMappings?api-version=" + loadTestingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateLoadTestProfileMapping",
			method: http.MethodPut,
			path:   parentBase + "/loadTestProfileMappings/profile-a?api-version=" + loadTestingAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"sourceResourceId":           parentResourceURI,
					"azureLoadTestingResourceId": loadTestID,
					"testProfileId":              "profile-a",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "PatchLoadTest",
			method: http.MethodPatch,
			path:   resourceBase + "?api-version=" + loadTestingAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"env":   "stage",
					"owner": "stackyard",
				},
				"properties": map[string]any{
					"description": "patched description",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteLoadTestMapping",
			method:   http.MethodDelete,
			path:     parentBase + "/loadTestMappings/mapping-a?api-version=" + loadTestingAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteLoadTestProfileMapping",
			method:   http.MethodDelete,
			path:     parentBase + "/loadTestProfileMappings/profile-a?api-version=" + loadTestingAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteLoadTest",
			method:   http.MethodDelete,
			path:     resourceBase + "?api-version=" + loadTestingAPIVersion,
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
