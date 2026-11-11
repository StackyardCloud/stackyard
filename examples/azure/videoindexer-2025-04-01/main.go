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

const videoIndexerAPIVersion = "2025-04-01"

type videoIndexerClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type videoIndexerSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *videoIndexerSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newVideoIndexerClient(endpoint, account, subscriptionKey string) *videoIndexerClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"videoindexer-2025-04-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&videoIndexerSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &videoIndexerClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *videoIndexerClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_VIDEOINDEXER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_VIDEOINDEXER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-videoindexer")
	accountName := getenv("STACKYARD_AZURE_VIDEOINDEXER_ACCOUNT_NAME", "video-account-a")

	fmt.Printf("Stackyard Azure Video Indexer example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newVideoIndexerClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	accountBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.VideoIndexer/accounts/" + accountName

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
			path:     "/azure/providers/Microsoft.VideoIndexer/operations?api-version=" + videoIndexerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path:   subscriptionBase + "/providers/Microsoft.VideoIndexer/checkNameAvailability?api-version=" + videoIndexerAPIVersion,
			payload: map[string]any{
				"name": accountName,
				"type": "Microsoft.VideoIndexer/accounts",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateAccount",
			method: http.MethodPut,
			path:   accountBase + "?api-version=" + videoIndexerAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"identity": map[string]any{"type": "SystemAssigned"},
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"description": "Local Video Indexer account",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetAccount",
			method:   http.MethodGet,
			path:     accountBase + "?api-version=" + videoIndexerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PatchAccount",
			method: http.MethodPatch,
			path:   accountBase + "?api-version=" + videoIndexerAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{"env": "stage"},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAccountsByResourceGroup",
			method:   http.MethodGet,
			path:     subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.VideoIndexer/accounts?api-version=" + videoIndexerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GenerateAccessToken",
			method: http.MethodPost,
			path:   accountBase + "/generateAccessToken?api-version=" + videoIndexerAPIVersion,
			payload: map[string]any{
				"permissionType":         "Reader",
				"scope":                  "Account",
				"tokenLifetimeInSeconds": 600,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GenerateRestrictedViewerAccessToken",
			method: http.MethodPost,
			path:   accountBase + "/generateRestrictedViewerAccessToken?api-version=" + videoIndexerAPIVersion,
			payload: map[string]any{
				"scope":                  "Video",
				"videoId":                "video-1",
				"tokenLifetimeInSeconds": 900,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GenerateExtensionAccessToken",
			method: http.MethodPost,
			path:   accountBase + "/generateExtensionAccessToken?api-version=" + videoIndexerAPIVersion,
			payload: map[string]any{
				"permissionType":         "Contributor",
				"scope":                  "Project",
				"projectId":              "project-1",
				"extensionId":            "ext-1",
				"tokenLifetimeInSeconds": 1200,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreatePrivateEndpointConnection",
			method: http.MethodPut,
			path:   accountBase + "/privateEndpointConnections/pec-a?api-version=" + videoIndexerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"privateEndpoint": map[string]any{"id": "/subscriptions/" + subscriptionID + "/resourceGroups/rg-network/providers/Microsoft.Network/privateEndpoints/pe-a"},
					"groupIds":        []any{"avi"},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListPrivateEndpointConnections",
			method:   http.MethodGet,
			path:     accountBase + "/privateEndpointConnections?api-version=" + videoIndexerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetPrivateLinkResource",
			method:   http.MethodGet,
			path:     accountBase + "/privateLinkResources/avi?api-version=" + videoIndexerAPIVersion,
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
