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

const graphServicesAPIVersion = "2023-04-13"

type graphServicesClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type graphServicesSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *graphServicesSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newGraphServicesClient(endpoint, account, subscriptionKey string) *graphServicesClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"graphservices-2023-04-13",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&graphServicesSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey},
			},
		},
		&policy.ClientOptions{},
	)
	return &graphServicesClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *graphServicesClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_GRAPHSERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_GRAPHSERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-graphservices")
	location := getenv("STACKYARD_AZURE_LOCATION", "global")

	resourceName := getenv("STACKYARD_AZURE_GRAPHSERVICES_RESOURCE_NAME", "graphaccount1")
	appID := getenv("STACKYARD_AZURE_GRAPHSERVICES_APP_ID", "11111111-aaaa-1111-bbbb-111111111111")

	client := newGraphServicesClient(endpoint, account, subscriptionKey)

	providerBase := "/azure/providers/Microsoft.GraphServices"
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	resourceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.GraphServices"
	accountBase := resourceBase + "/accounts/" + resourceName

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "OperationsList",
			method:   http.MethodGet,
			path:     providerBase + "/operations?api-version=" + graphServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "AccountsCreateAndUpdate",
			method: http.MethodPut,
			path:   accountBase + "?api-version=" + graphServicesAPIVersion,
			payload: map[string]any{
				"location": location,
				"tags": map[string]any{
					"environment": "local",
					"service":     "graphservices",
				},
				"properties": map[string]any{
					"appId": appID,
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "AccountsGet",
			method:   http.MethodGet,
			path:     accountBase + "?api-version=" + graphServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AccountsListByResourceGroup",
			method:   http.MethodGet,
			path:     resourceBase + "/accounts?api-version=" + graphServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AccountsListBySubscription",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.GraphServices/accounts?api-version=" + graphServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "AccountsUpdate",
			method: http.MethodPatch,
			path:   accountBase + "?api-version=" + graphServicesAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"environment": "local",
					"updated":     "true",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AccountsDelete",
			method:   http.MethodDelete,
			path:     accountBase + "?api-version=" + graphServicesAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
	}

	for _, call := range calls {
		response, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, response)
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
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("%s -> %d\n%s\n", name, status, string(formatted))
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
