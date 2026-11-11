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

const cdnAPIVersion = "2025-04-15"

type cdnClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type cdnSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *cdnSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCDNClient(endpoint, account, subscriptionKey string) *cdnClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"cdn-2025-04-15",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&cdnSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &cdnClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *cdnClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_CDN_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CDN_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-cdn")
	profileName := getenv("STACKYARD_AZURE_CDN_PROFILE_NAME", "profile-a")
	endpointName := getenv("STACKYARD_AZURE_CDN_ENDPOINT_NAME", "endpoint-a")

	fmt.Printf("Stackyard Azure CDN example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCDNClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	profileBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Cdn/profiles/" + profileName
	endpointBase := profileBase + "/endpoints/" + endpointName

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
			path:     "/azure/providers/Microsoft.Cdn/operations?api-version=" + cdnAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path:   "/azure/providers/Microsoft.Cdn/checkNameAvailability?api-version=" + cdnAPIVersion,
			payload: map[string]any{
				"name": endpointName,
				"type": "Microsoft.Cdn/Profiles/Endpoints",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateProfile",
			method: http.MethodPut,
			path:   profileBase + "?api-version=" + cdnAPIVersion,
			payload: map[string]any{
				"location": "global",
				"tags":     map[string]any{"env": "dev"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateEndpoint",
			method: http.MethodPut,
			path:   endpointBase + "?api-version=" + cdnAPIVersion,
			payload: map[string]any{
				"location": "global",
				"properties": map[string]any{
					"origins": []map[string]any{
						{
							"name":       "origin-inline",
							"properties": map[string]any{"hostName": "origin.example.com", "httpPort": 80, "httpsPort": 443},
						},
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetSupportedOptimizationTypes",
			method:   http.MethodPost,
			path:     profileBase + "/getSupportedOptimizationTypes?api-version=" + cdnAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ValidateProbe",
			method:   http.MethodPost,
			path:     subscriptionBase + "/providers/Microsoft.Cdn/validateProbe?api-version=" + cdnAPIVersion,
			payload:  map[string]any{"probeURL": "https://example.com/healthz"},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrigin",
			method:   http.MethodPut,
			path:     endpointBase + "/origins/origin-a?api-version=" + cdnAPIVersion,
			payload:  map[string]any{"properties": map[string]any{"hostName": "origin-a.example.com", "httpPort": 80, "httpsPort": 443}},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "CreateCustomDomain",
			method:   http.MethodPut,
			path:     endpointBase + "/customDomains/domain-a?api-version=" + cdnAPIVersion,
			payload:  map[string]any{"properties": map[string]any{"hostName": "cdn.example.com"}},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ValidateCustomDomain",
			method:   http.MethodPost,
			path:     endpointBase + "/validateCustomDomain?api-version=" + cdnAPIVersion,
			payload:  map[string]any{"hostName": "cdn.example.com"},
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
