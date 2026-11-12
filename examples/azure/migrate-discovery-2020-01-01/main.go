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

const migrateDiscoveryAPIVersion = "2020-01-01"

type migrateDiscoveryClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type migrateDiscoverySharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *migrateDiscoverySharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newMigrateDiscoveryClient(endpoint, account, subscriptionKey string) *migrateDiscoveryClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"migrate-discovery-2020-01-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&migrateDiscoverySharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &migrateDiscoveryClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *migrateDiscoveryClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	var payloadMap map[string]any
	if err := json.Unmarshal(body, &payloadMap); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	return payloadMap, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_MIGRATE_DISCOVERY_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_MIGRATE_DISCOVERY_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-discovery")
	vmwareSite := getenv("STACKYARD_AZURE_MIGRATE_DISCOVERY_VMWARE_SITE", "vmw-site")
	hypervSite := getenv("STACKYARD_AZURE_MIGRATE_DISCOVERY_HYPERV_SITE", "hv-site")

	fmt.Printf("Stackyard Azure Migrate Discovery example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newMigrateDiscoveryClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.OffAzure"

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
			path:     "/azure/providers/Microsoft.OffAzure/operations?api-version=" + migrateDiscoveryAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateVmwareSite",
			method:   http.MethodPut,
			path:     base + "/VMwareSites/" + vmwareSite + "?api-version=" + migrateDiscoveryAPIVersion,
			payload:  map[string]any{"location": "eastus", "tags": map[string]any{"env": "dev"}},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListVmwareMachines",
			method:   http.MethodGet,
			path:     base + "/VMwareSites/" + vmwareSite + "/machines?api-version=" + migrateDiscoveryAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateHyperVSite",
			method:   http.MethodPut,
			path:     base + "/HyperVSites/" + hypervSite + "?api-version=" + migrateDiscoveryAPIVersion,
			payload:  map[string]any{"location": "eastus2", "tags": map[string]any{"env": "test"}},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetHyperVHost",
			method:   http.MethodGet,
			path:     base + "/HyperVSites/" + hypervSite + "/hosts/host-1?api-version=" + migrateDiscoveryAPIVersion,
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
