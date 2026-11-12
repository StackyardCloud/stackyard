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

const dataManagerForAgriAPIVersion = "2023-06-01-preview"

type dataManagerForAgriClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dataManagerForAgriSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dataManagerForAgriSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDataManagerForAgriClient(endpoint, account, subscriptionKey string) *dataManagerForAgriClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"data-manager-for-agri-resourcemanager-2023-06-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&dataManagerForAgriSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &dataManagerForAgriClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataManagerForAgriClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, http.Header, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, nil, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("read response %s %s: %w", method, path, readErr)
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
			return nil, resp.StatusCode, resp.Header, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, resp.Header, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, resp.Header, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_AGRI_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AGRI_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-agri")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	farmBeatName := getenv("STACKYARD_AZURE_FARMBEATS_NAME", "farm-a")

	fmt.Printf("Stackyard Azure Data Manager for Agriculture example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newDataManagerForAgriClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	farmBeatBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AgFoodPlatform/farmBeats/" + farmBeatName
	dataConnectorBase := farmBeatBase + "/dataConnectors/connector-a"

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
			path:     "/azure/providers/Microsoft.AgFoodPlatform/operations?api-version=" + dataManagerForAgriAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path:   subscriptionBase + "/providers/Microsoft.AgFoodPlatform/checkNameAvailability?api-version=" + dataManagerForAgriAPIVersion,
			payload: map[string]any{
				"name": farmBeatName,
				"type": "Microsoft.AgFoodPlatform/farmBeats",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateFarmBeats",
			method: http.MethodPut,
			path:   farmBeatBase + "?api-version=" + dataManagerForAgriAPIVersion,
			payload: map[string]any{
				"location": region,
				"tags":     map[string]any{"env": "dev"},
				"identity": map[string]any{"type": "SystemAssigned"},
				"properties": map[string]any{
					"publicNetworkAccess": "Enabled",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetFarmBeats",
			method:   http.MethodGet,
			path:     farmBeatBase + "?api-version=" + dataManagerForAgriAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListExtensionDefinitions",
			method:   http.MethodGet,
			path:     farmBeatBase + "/farmBeatsExtensionDefinitions?api-version=" + dataManagerForAgriAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateDataConnector",
			method: http.MethodPut,
			path:   dataConnectorBase + "?api-version=" + dataManagerForAgriAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"credentials": map[string]any{
						"kind":         "OAuth",
						"clientId":     "stackyard-client",
						"clientSecret": "stackyard-secret",
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListPrivateLinkResources",
			method:   http.MethodGet,
			path:     farmBeatBase + "/privateLinkResources?api-version=" + dataManagerForAgriAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteDataConnector",
			method:   http.MethodDelete,
			path:     dataConnectorBase + "?api-version=" + dataManagerForAgriAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteFarmBeats",
			method:   http.MethodDelete,
			path:     farmBeatBase + "?api-version=" + dataManagerForAgriAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
	}

	for _, call := range calls {
		payload, status, headers, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, headers.Get("Azure-AsyncOperation"), payload)
	}
}

func printJSON(name string, status int, async string, payload map[string]any) {
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n", name, status)
	if strings.TrimSpace(async) != "" {
		fmt.Printf("Azure-AsyncOperation: %s\n", async)
	}
	fmt.Printf("%s\n", string(formatted))
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
