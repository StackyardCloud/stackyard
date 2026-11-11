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

const springAppsAPIVersion = "2023-12-01"

type springAppsClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type springAppsSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *springAppsSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newSpringAppsClient(endpoint, account, subscriptionKey string) *springAppsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"azurespringapps-2023-12-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&springAppsSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &springAppsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *springAppsClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_SPRING_APPS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SPRING_APPS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-spring")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	serviceName := getenv("STACKYARD_AZURE_SPRING_SERVICE_NAME", "spring-a")
	appName := getenv("STACKYARD_AZURE_SPRING_APP_NAME", "app-a")
	deploymentName := getenv("STACKYARD_AZURE_SPRING_DEPLOYMENT_NAME", "deployment-a")

	client := newSpringAppsClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	serviceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppPlatform/Spring/" + serviceName
	appBase := serviceBase + "/apps/" + appName
	buildServiceBase := serviceBase + "/buildServices/default"

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{"ListOperations", http.MethodGet, "/azure/providers/Microsoft.AppPlatform/operations?api-version=" + springAppsAPIVersion, nil, []int{http.StatusOK}},
		{"CreateService", http.MethodPut, serviceBase + "?api-version=" + springAppsAPIVersion, map[string]any{
			"location": location,
			"tags":     map[string]any{"env": "dev"},
			"properties": map[string]any{
				"networkProfile": "default",
			},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"EnableTestEndpoint", http.MethodPost, serviceBase + "/enableTestEndpoint?api-version=" + springAppsAPIVersion, map[string]any{}, []int{http.StatusOK}},
		{"CreateApp", http.MethodPut, appBase + "?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"public": true},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"CreateDeployment", http.MethodPut, appBase + "/deployments/" + deploymentName + "?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"source": map[string]any{"type": "Jar", "relativePath": "app.jar"}},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"SetActiveDeployment", http.MethodPost, appBase + "/setActiveDeployments?api-version=" + springAppsAPIVersion, map[string]any{
			"deploymentName": deploymentName,
		}, []int{http.StatusOK}},
		{"CreateBinding", http.MethodPut, appBase + "/bindings/binding-a?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"resourceId": "/subscriptions/mock/resourceGroups/mock/providers/Microsoft.Storage/storageAccounts/storagea"},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"CreateCustomDomain", http.MethodPut, appBase + "/domains/domain-a?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"certName": "cert-a"},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"CreateCertificate", http.MethodPut, serviceBase + "/certificates/cert-a?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"issuer": "Stackyard CA"},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"CreateContainerRegistry", http.MethodPut, serviceBase + "/containerRegistries/registry-a?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"server": "registry.azurecr.io"},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"ValidateContainerRegistry", http.MethodPost, serviceBase + "/containerRegistries/registry-a/validate?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"server": "registry.azurecr.io"},
		}, []int{http.StatusOK}},
		{"CreateGateway", http.MethodPut, serviceBase + "/gateways/gateway-a?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"public": true},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"CreateAPIPortal", http.MethodPut, serviceBase + "/apiPortals/portal-a?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"url": "https://api-portal.stackyard.local"},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"CreateConfigurationService", http.MethodPut, serviceBase + "/configurationServices/config-a?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"generation": "Gen2"},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"ValidateConfigurationService", http.MethodPost, serviceBase + "/configurationServices/config-a/validate?api-version=" + springAppsAPIVersion, map[string]any{}, []int{http.StatusOK}},
		{"CreateBuildService", http.MethodPut, buildServiceBase + "?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"resourceRequests": map[string]any{"cpu": "1", "memory": "2Gi"}},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"CreateBuild", http.MethodPut, buildServiceBase + "/builds/build-a?api-version=" + springAppsAPIVersion, map[string]any{
			"properties": map[string]any{"builderName": "default"},
		}, []int{http.StatusCreated, http.StatusOK}},
		{"ListBuildResults", http.MethodGet, buildServiceBase + "/buildResults?api-version=" + springAppsAPIVersion, nil, []int{http.StatusOK}},
		{"ListSupportedBuildpacks", http.MethodGet, buildServiceBase + "/supportedBuildpacks?api-version=" + springAppsAPIVersion, nil, []int{http.StatusOK}},
		{"ListSupportedStacks", http.MethodGet, buildServiceBase + "/supportedStacks?api-version=" + springAppsAPIVersion, nil, []int{http.StatusOK}},
		{"ListBuildpackBindings", http.MethodGet, buildServiceBase + "/builders/default/buildpackBindings?api-version=" + springAppsAPIVersion, nil, []int{http.StatusOK}},
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
		fatalf("marshal %s response: %v", name, err)
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
