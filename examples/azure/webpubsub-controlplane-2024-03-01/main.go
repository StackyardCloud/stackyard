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

const webPubSubAPIVersion = "2024-03-01"

type webPubSubClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type webPubSubSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *webPubSubSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newWebPubSubClient(endpoint, account, subscriptionKey string) *webPubSubClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"webpubsub-controlplane-2024-03-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&webPubSubSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &webPubSubClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *webPubSubClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_WEBPUBSUB_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_WEBPUBSUB_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-webpubsub")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	resourceName := getenv("STACKYARD_AZURE_WEBPUBSUB_NAME", "webpubsub-a")

	fmt.Printf("Stackyard Azure Web PubSub control-plane example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newWebPubSubClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	accountBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.SignalRService/webPubSub/" + resourceName

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
			path:     "/azure/providers/Microsoft.SignalRService/operations?api-version=" + webPubSubAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListUsages",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.SignalRService/locations/" + location + "/usages?api-version=" + webPubSubAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path:   subscriptionBase + "/providers/Microsoft.SignalRService/locations/" + location + "/checkNameAvailability?api-version=" + webPubSubAPIVersion,
			payload: map[string]any{
				"name": resourceName,
				"type": "Microsoft.SignalRService/webPubSub",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateAccount",
			method: http.MethodPut,
			path:   accountBase + "?api-version=" + webPubSubAPIVersion,
			payload: map[string]any{
				"location": location,
				"identity": map[string]any{"type": "SystemAssigned"},
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"publicNetworkAccess": "Enabled",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetAccount",
			method:   http.MethodGet,
			path:     accountBase + "?api-version=" + webPubSubAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PatchAccount",
			method: http.MethodPatch,
			path:   accountBase + "?api-version=" + webPubSubAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{"env": "stage"},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListKeys",
			method:   http.MethodPost,
			path:     accountBase + "/listKeys?api-version=" + webPubSubAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "RegenerateKey",
			method: http.MethodPost,
			path:   accountBase + "/regenerateKey?api-version=" + webPubSubAPIVersion,
			payload: map[string]any{
				"keyType": "Primary",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSkus",
			method:   http.MethodGet,
			path:     accountBase + "/skus?api-version=" + webPubSubAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateCustomCertificate",
			method: http.MethodPut,
			path:   accountBase + "/customCertificates/cert-a?api-version=" + webPubSubAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"keyVaultBaseUri": "https://stackyard.vault.azure.net/"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateHub",
			method: http.MethodPut,
			path:   accountBase + "/hubs/hub-a?api-version=" + webPubSubAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"anonymousConnectPolicy": "deny"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateReplica",
			method: http.MethodPut,
			path:   accountBase + "/replicas/replica-a?api-version=" + webPubSubAPIVersion,
			payload: map[string]any{
				"location": "westus2",
				"properties": map[string]any{
					"regionEndpointEnabled": true,
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListReplicaSkus",
			method:   http.MethodGet,
			path:     accountBase + "/replicas/replica-a/skus?api-version=" + webPubSubAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateLinkResources",
			method:   http.MethodGet,
			path:     accountBase + "/privateLinkResources?api-version=" + webPubSubAPIVersion,
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
