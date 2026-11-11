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

const confluentAPIVersion = "2024-07-01"

type confluentClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type confluentSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *confluentSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newConfluentClient(endpoint, account, subscriptionKey string) *confluentClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"confluent-2024-07-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&confluentSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &confluentClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *confluentClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_CONFLUENT_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CONFLUENT_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-confluent")
	organizationName := getenv("STACKYARD_AZURE_CONFLUENT_ORGANIZATION", "stackyard-confluent")
	environmentID := getenv("STACKYARD_AZURE_CONFLUENT_ENVIRONMENT", "env-dev")
	clusterID := getenv("STACKYARD_AZURE_CONFLUENT_CLUSTER", "lkc-dev")

	client := newConfluentClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	orgBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Confluent/organizations/" + organizationName
	clusterBase := orgBase + "/environments/" + environmentID + "/clusters/" + clusterID

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
			path:     "/azure/providers/Microsoft.Confluent/operations?api-version=" + confluentAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "AcceptMarketplaceAgreement",
			method: http.MethodPut,
			path:   subscriptionBase + "/providers/Microsoft.Confluent/agreements/default?api-version=" + confluentAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"accepted":  true,
					"plan":      "enterprise",
					"product":   "AzureConfluent",
					"publisher": "Microsoft",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateOrganization",
			method: http.MethodPut,
			path:   orgBase + "?api-version=" + confluentAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"offerDetail": map[string]any{
						"planId":      "enterprise",
						"publisherId": "microsoft",
					},
					"userDetail": map[string]any{
						"firstName":    "Stack",
						"lastName":     "Yard",
						"emailAddress": "owner@example.com",
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateEnvironment",
			method: http.MethodPut,
			path:   orgBase + "/environments/" + environmentID + "?api-version=" + confluentAPIVersion,
			payload: map[string]any{
				"location": "global",
				"properties": map[string]any{
					"kind": "StreamGovernance",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateCluster",
			method: http.MethodPut,
			path:   clusterBase + "?api-version=" + confluentAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"properties": map[string]any{
					"region": "eastus",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateTopic",
			method: http.MethodPut,
			path:   clusterBase + "/topics/orders?api-version=" + confluentAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"properties": map[string]any{
					"partitionsCount":   6,
					"replicationFactor": 3,
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListClustersAccessView",
			method:   http.MethodPost,
			path:     orgBase + "/access/default/listClusters?api-version=" + confluentAPIVersion,
			payload:  map[string]any{"searchFilters": map[string]any{}},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ValidateOrganization",
			method: http.MethodPost,
			path:   subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Confluent/validations/" + organizationName + "/orgValidate?api-version=" + confluentAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"properties": map[string]any{
					"offerDetail": map[string]any{"planId": "enterprise"},
				},
			},
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
