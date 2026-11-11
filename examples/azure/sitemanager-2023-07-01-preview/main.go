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

const siteManagerAPIVersion = "2023-07-01-preview"

type siteManagerClient struct {
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

func newSiteManagerClient(endpoint, account, subscriptionKey string) *siteManagerClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"sitemanager-2023-07-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &siteManagerClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *siteManagerClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_SITEMANAGER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SITEMANAGER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-site-manager")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	siteName := getenv("STACKYARD_AZURE_SITEMANAGER_SITE", "site-a")

	fmt.Printf("Stackyard Azure Arc Site Manager (sitemanager-2023-07-01-preview) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newSiteManagerClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	siteScope := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Edge/sites/" + siteName

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
			path:     "/azure/providers/Microsoft.Edge/operations?api-version=" + siteManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateSite",
			method: http.MethodPut,
			path:   siteScope + "?api-version=" + siteManagerAPIVersion,
			payload: map[string]any{
				"location": region,
				"kind":     "Arc",
				"tags":     map[string]any{"env": "dev"},
				"extendedLocation": map[string]any{
					"type": "CustomLocation",
					"name": "custom-eastus",
				},
				"identity": map[string]any{"type": "SystemAssigned"},
				"properties": map[string]any{
					"displayName":                   "Stackyard Arc Site",
					"description":                   "local site manager example",
					"servicePrincipalApplicationId": "11111111-1111-1111-1111-111111111111",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetSite",
			method:   http.MethodGet,
			path:     siteScope + "?api-version=" + siteManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSitesByResourceGroup",
			method:   http.MethodGet,
			path:     base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Edge/sites?api-version=" + siteManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSitesBySubscription",
			method:   http.MethodGet,
			path:     base + "/providers/Microsoft.Edge/sites?api-version=" + siteManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PatchSite",
			method: http.MethodPatch,
			path:   siteScope + "?api-version=" + siteManagerAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{"env": "stage", "owner": "stackyard"},
				"properties": map[string]any{
					"description": "updated local site manager example",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteSite",
			method:   http.MethodDelete,
			path:     siteScope + "?api-version=" + siteManagerAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
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
