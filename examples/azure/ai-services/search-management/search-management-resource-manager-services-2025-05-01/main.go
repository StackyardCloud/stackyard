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

const searchManagementServicesAPIVersion = "2025-05-01"

type searchManagementServicesClient struct {
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

func newSearchManagementServicesClient(endpoint, account, subscriptionKey string) *searchManagementServicesClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"search-management-resource-manager-services-2025-05-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &searchManagementServicesClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *searchManagementServicesClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

	if payload != nil {
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

	if len(expectedStatuses) == 0 {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else {
		matched := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				matched = true
				break
			}
		}
		if !matched {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
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
	account := getenv("STACKYARD_AZURE_SEARCH_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SEARCH_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-search")
	searchServiceName := getenv("STACKYARD_AZURE_SEARCH_SERVICE_NAME", "my-search")

	fmt.Printf("Stackyard Azure Search Management - Resource Manager - Services (search-management-resource-manager-services-2025-05-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newSearchManagementServicesClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	resourceBase := subscriptionBase +
		"/resourceGroups/" + resourceGroup +
		"/providers/Microsoft.Search/searchServices/" + searchServiceName

	createPayload := map[string]any{
		"location": "eastus",
		"sku": map[string]any{
			"name": "basic",
		},
		"properties": map[string]any{
			"replicaCount":   1,
			"partitionCount": 1,
		},
	}
	updatePayload := map[string]any{
		"tags": map[string]any{
			"env": "test",
		},
	}
	checkNamePayload := map[string]any{
		"name": searchServiceName,
		"type": "searchServices",
	}

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "CheckNameAvailability",
			method:   http.MethodPost,
			path:     subscriptionBase + "/providers/Microsoft.Search/checkNameAvailability",
			payload:  checkNamePayload,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "CreateOrUpdate",
			method:   http.MethodPut,
			path:     resourceBase,
			payload:  createPayload,
			statuses: []int{http.StatusCreated, http.StatusOK, http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:     "Get",
			method:   http.MethodGet,
			path:     resourceBase,
			statuses: []int{http.StatusOK, http.StatusNotFound, http.StatusNotImplemented},
		},
		{
			name:     "Update",
			method:   http.MethodPatch,
			path:     resourceBase,
			payload:  updatePayload,
			statuses: []int{http.StatusOK, http.StatusNotFound, http.StatusNotImplemented},
		},
		{
			name:     "ListByResourceGroup",
			method:   http.MethodGet,
			path:     subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Search/searchServices",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListBySubscription",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.Search/searchServices",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "Upgrade",
			method:   http.MethodPost,
			path:     resourceBase + "/upgrade",
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:     "Delete",
			method:   http.MethodDelete,
			path:     resourceBase,
			statuses: []int{http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound, http.StatusNotImplemented},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		fullPath := withAPIVersion(call.path, searchManagementServicesAPIVersion)
		_, status, err := client.doJSON(ctx, call.method, fullPath, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		if status == http.StatusNotImplemented {
			notImplementedCount++
			notImplemented(fullPath)
			continue
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All search-management resource-manager services routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
}

func withAPIVersion(path, apiVersion string) string {
	if strings.Contains(path, "?") {
		return path + "&api-version=" + apiVersion
	}
	return path + "?api-version=" + apiVersion
}

func notImplemented(path string) {
	fmt.Printf("Route is recognized but not implemented yet: %s\n", path)
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
