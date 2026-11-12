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

const cloudNGFWAPIVersion = "2025-10-08"

type cloudNGFWClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type cloudNGFWSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *cloudNGFWSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCloudNGFWClient(endpoint, account, subscriptionKey string) *cloudNGFWClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"cloudngfw-2025-10-08",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&cloudNGFWSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &cloudNGFWClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *cloudNGFWClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_CLOUDNGFW_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CLOUDNGFW_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-ngfw")
	firewallName := getenv("STACKYARD_AZURE_CLOUDNGFW_FIREWALL", "fw-a")
	localRulestackName := getenv("STACKYARD_AZURE_CLOUDNGFW_LOCAL_RULESTACK", "local-a")
	globalRulestackName := getenv("STACKYARD_AZURE_CLOUDNGFW_GLOBAL_RULESTACK", "global-a")

	fmt.Printf("Stackyard Azure Cloud NGFW example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCloudNGFWClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	firewallBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/PaloAltoNetworks.Cloudngfw/firewalls/" + firewallName
	localBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/PaloAltoNetworks.Cloudngfw/localRulestacks/" + localRulestackName
	globalBase := "/azure/providers/PaloAltoNetworks.Cloudngfw/globalRulestacks/" + globalRulestackName

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
			path:     "/azure/providers/PaloAltoNetworks.Cloudngfw/operations?api-version=" + cloudNGFWAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateFirewall",
			method: http.MethodPut,
			path:   firewallBase + "?api-version=" + cloudNGFWAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"properties": map[string]any{
					"localRulestackName":  localRulestackName,
					"globalRulestackName": globalRulestackName,
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetFirewallStatus",
			method:   http.MethodGet,
			path:     firewallBase + "/statuses/default?api-version=" + cloudNGFWAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateLocalRulestack",
			method: http.MethodPut,
			path:   localBase + "?api-version=" + cloudNGFWAPIVersion,
			payload: map[string]any{
				"location": "eastus",
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateLocalFQDNList",
			method: http.MethodPut,
			path:   localBase + "/fqdnlists/fqdn-a?api-version=" + cloudNGFWAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"properties": map[string]any{
					"fqdnList": []any{"example.com"},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "CommitLocalRulestack",
			method:   http.MethodPost,
			path:     localBase + "/commit?api-version=" + cloudNGFWAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateGlobalRulestack",
			method: http.MethodPut,
			path:   globalBase + "?api-version=" + cloudNGFWAPIVersion,
			payload: map[string]any{
				"location": "global",
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateGlobalPrefixList",
			method: http.MethodPut,
			path:   globalBase + "/prefixlists/prefix-a?api-version=" + cloudNGFWAPIVersion,
			payload: map[string]any{
				"location": "global",
				"properties": map[string]any{
					"prefixes": []any{"10.0.0.0/24"},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListGlobalRulestacks",
			method:   http.MethodGet,
			path:     "/azure/providers/PaloAltoNetworks.Cloudngfw/globalRulestacks?api-version=" + cloudNGFWAPIVersion,
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
