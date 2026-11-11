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

const azureStackHCIAPIVersion = "2024-04-01"

type azureStackHCIClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type azureStackHCISharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *azureStackHCISharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newAzureStackHCIClient(endpoint, account, subscriptionKey string) *azureStackHCIClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"stackhci-2024-04-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&azureStackHCISharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &azureStackHCIClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *azureStackHCIClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_STACKHCI_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_STACKHCI_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-stackhci")
	clusterName := getenv("STACKYARD_AZURE_STACKHCI_CLUSTER", "cluster-a")

	fmt.Printf("Stackyard Azure Stack HCI example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAzureStackHCIClient(endpoint, account, subscriptionKey)
	clusterBase := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureStackHCI/clusters/" + clusterName
	arcBase := clusterBase + "/arcSettings/default"
	extensionBase := arcBase + "/extensions/telemetry"
	deploymentBase := clusterBase + "/deploymentSettings/default"
	updateBase := clusterBase + "/updates/update-1"
	clusterURI := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureStackHCI/clusters/" + clusterName
	edgeBase := "/azure" + clusterURI + "/providers/Microsoft.AzureStackHCI/edgeDevices/edge-a"

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
			path:     "/azure/providers/Microsoft.AzureStackHCI/operations?api-version=" + azureStackHCIAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateCluster",
			method: http.MethodPut,
			path:   clusterBase + "?api-version=" + azureStackHCIAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"identity": map[string]any{"type": "SystemAssigned"},
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"desiredProperties": map[string]any{"diagnosticLevel": "Enhanced"},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ConfigureRemoteSupport",
			method:   http.MethodPost,
			path:     clusterBase + "/configureRemoteSupport?api-version=" + azureStackHCIAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateArcSetting",
			method: http.MethodPut,
			path:   arcBase + "?api-version=" + azureStackHCIAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"connectivityState": "Connected"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "CreateArcIdentity",
			method:   http.MethodPost,
			path:     arcBase + "/createIdentity?api-version=" + azureStackHCIAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateExtension",
			method: http.MethodPut,
			path:   extensionBase + "?api-version=" + azureStackHCIAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"version": "1.0.0"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "UpgradeExtension",
			method:   http.MethodPost,
			path:     extensionBase + "/upgrade?api-version=" + azureStackHCIAPIVersion,
			payload:  map[string]any{"targetVersion": "1.1.0"},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateDeploymentSettings",
			method: http.MethodPut,
			path:   deploymentBase + "?api-version=" + azureStackHCIAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"deploymentMode": "ValidateOnly", "deploymentIntent": "Deploy"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateUpdate",
			method: http.MethodPut,
			path:   updateBase + "?api-version=" + azureStackHCIAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"packageVersion": "2404.0", "state": "Available"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ApplyUpdate",
			method:   http.MethodPost,
			path:     updateBase + "?api-version=" + azureStackHCIAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetUpdateSummary",
			method:   http.MethodGet,
			path:     clusterBase + "/updateSummaries/default?api-version=" + azureStackHCIAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetUpdateRun",
			method:   http.MethodGet,
			path:     updateBase + "/updateRuns/run-latest?api-version=" + azureStackHCIAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateEdgeDevice",
			method: http.MethodPut,
			path:   edgeBase + "?api-version=" + azureStackHCIAPIVersion,
			payload: map[string]any{
				"kind":       "HCI",
				"properties": map[string]any{"deviceType": "IndustrialGateway"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ValidateEdgeDevice",
			method:   http.MethodPost,
			path:     edgeBase + "/validate?api-version=" + azureStackHCIAPIVersion,
			payload:  map[string]any{},
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
