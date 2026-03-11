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

const apiCenterResourceManagerAPIVersion = "2024-03-01"

type apiCenterClient struct {
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

func newAPICenterClient(endpoint, account, subscriptionKey string) *apiCenterClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"api-center-resource-manager-2024-03-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &apiCenterClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *apiCenterClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

	if len(expectedStatuses) == 0 {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else {
		matched := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				matched = true
				break
			}
		}
		if !matched {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
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
	account := getenv("STACKYARD_AZURE_APICENTER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APICENTER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-apic")
	serviceName := getenv("STACKYARD_AZURE_APICENTER_SERVICE", "contoso")
	workspaceName := getenv("STACKYARD_AZURE_APICENTER_WORKSPACE", "default")
	apiName := getenv("STACKYARD_AZURE_APICENTER_API", "echo-api")
	versionName := getenv("STACKYARD_AZURE_APICENTER_VERSION", "2023-01-01")
	definitionName := getenv("STACKYARD_AZURE_APICENTER_DEFINITION", "openapi")
	deploymentName := getenv("STACKYARD_AZURE_APICENTER_DEPLOYMENT", "production")
	environmentName := getenv("STACKYARD_AZURE_APICENTER_ENVIRONMENT", "public")
	metadataSchemaName := getenv("STACKYARD_AZURE_APICENTER_METADATA_SCHEMA", "author")

	fmt.Printf("Stackyard Azure API Center Resource Manager (resource-manager-2024-03-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAPICenterClient(endpoint, account, subscriptionKey)

	serviceScope := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ApiCenter/services/" + serviceName
	workspaceScope := serviceScope + "/workspaces/" + workspaceName
	apiScope := workspaceScope + "/apis/" + apiName
	versionScope := apiScope + "/versions/" + versionName
	definitionScope := versionScope + "/definitions/" + definitionName
	deploymentScope := apiScope + "/deployments/" + deploymentName
	environmentScope := workspaceScope + "/environments/" + environmentName
	metadataSchemaScope := serviceScope + "/metadataSchemas/" + metadataSchemaName

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
			path:     "/azure/providers/Microsoft.ApiCenter/operations?api-version=" + apiCenterResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateService",
			method: http.MethodPut,
			path:   serviceScope + "?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"sku": map[string]any{
					"name": "Standard",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetService",
			method:   http.MethodGet,
			path:     serviceScope + "?api-version=" + apiCenterResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "HeadService",
			method:   http.MethodHead,
			path:     serviceScope + "?api-version=" + apiCenterResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateService",
			method: http.MethodPatch,
			path:   serviceScope + "?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"env": "dev",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListServicesByResourceGroup",
			method:   http.MethodGet,
			path:     "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ApiCenter/services?api-version=" + apiCenterResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListServicesBySubscription",
			method:   http.MethodGet,
			path:     "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.ApiCenter/services?api-version=" + apiCenterResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ExportMetadataSchema",
			method: http.MethodPost,
			path:   serviceScope + "/exportMetadataSchema?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"assignedTo": "api",
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "CreateWorkspace",
			method: http.MethodPut,
			path:   workspaceScope + "?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"title": workspaceName},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:   "CreateAPI",
			method: http.MethodPut,
			path:   apiScope + "?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"title": apiName},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:   "CreateAPIVersion",
			method: http.MethodPut,
			path:   versionScope + "?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"title": versionName},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:   "CreateAPIDefinition",
			method: http.MethodPut,
			path:   definitionScope + "?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"title": definitionName},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:   "ExportSpecification",
			method: http.MethodPost,
			path:   definitionScope + "/exportSpecification?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"format": "openapi",
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "ImportSpecification",
			method: http.MethodPost,
			path:   definitionScope + "/importSpecification?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"value": "{\"openapi\":\"3.0.0\"}",
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "CreateDeployment",
			method: http.MethodPut,
			path:   deploymentScope + "?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"title": deploymentName},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:   "CreateEnvironment",
			method: http.MethodPut,
			path:   environmentScope + "?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"title": environmentName},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:   "CreateMetadataSchema",
			method: http.MethodPut,
			path:   metadataSchemaScope + "?api-version=" + apiCenterResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"schema": map[string]any{"type": "object"},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
	}

	for _, call := range calls {
		_, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	fmt.Println("Azure API Center Resource Manager staged routes completed successfully.")
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
