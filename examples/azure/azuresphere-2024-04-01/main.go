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

const azureSphereAPIVersion = "2024-04-01"

type azureSphereClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type azureSphereSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *azureSphereSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newAzureSphereClient(endpoint, account, subscriptionKey string) *azureSphereClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"azuresphere-2024-04-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&azureSphereSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &azureSphereClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *azureSphereClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_SPHERE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SPHERE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-sphere")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	catalogName := getenv("STACKYARD_AZURE_SPHERE_CATALOG", "catalog-a")
	productName := getenv("STACKYARD_AZURE_SPHERE_PRODUCT", "product-a")
	deviceGroupName := getenv("STACKYARD_AZURE_SPHERE_DEVICE_GROUP", "group-a")
	deviceName := getenv("STACKYARD_AZURE_SPHERE_DEVICE", strings.Repeat("0", 128))
	deploymentName := getenv("STACKYARD_AZURE_SPHERE_DEPLOYMENT", "deployment-a")
	imageName := getenv("STACKYARD_AZURE_SPHERE_IMAGE", "image-a")

	fmt.Printf("Stackyard Azure Sphere example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAzureSphereClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	catalogBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureSphere/catalogs/" + catalogName
	productBase := catalogBase + "/products/" + productName
	deviceGroupBase := productBase + "/deviceGroups/" + deviceGroupName

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
			path:     "/azure/providers/Microsoft.AzureSphere/operations?api-version=" + azureSphereAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateCatalog",
			method: http.MethodPut,
			path:   catalogBase + "?api-version=" + azureSphereAPIVersion,
			payload: map[string]any{
				"location": region,
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"description": "Stackyard Azure Sphere catalog",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetCatalog",
			method:   http.MethodGet,
			path:     catalogBase + "?api-version=" + azureSphereAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListCatalogsByResourceGroup",
			method:   http.MethodGet,
			path:     subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureSphere/catalogs?api-version=" + azureSphereAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateProduct",
			method: http.MethodPut,
			path:   productBase + "?api-version=" + azureSphereAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"description": "Stackyard Azure Sphere product",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GenerateDefaultDeviceGroups",
			method:   http.MethodPost,
			path:     productBase + "/generateDefaultDeviceGroups?api-version=" + azureSphereAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateDeviceGroup",
			method: http.MethodPut,
			path:   deviceGroupBase + "?api-version=" + azureSphereAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"description": "Stackyard Azure Sphere device group",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "ClaimDevices",
			method: http.MethodPost,
			path:   deviceGroupBase + "/claimDevices?api-version=" + azureSphereAPIVersion,
			payload: map[string]any{
				"deviceIdentifiers": []any{deviceName},
			},
			statuses: []int{http.StatusAccepted},
		},
		{
			name:     "CountDevices",
			method:   http.MethodPost,
			path:     deviceGroupBase + "/countDevices?api-version=" + azureSphereAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDevices",
			method:   http.MethodGet,
			path:     deviceGroupBase + "/devices?api-version=" + azureSphereAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateDeployment",
			method: http.MethodPut,
			path:   deviceGroupBase + "/deployments/" + deploymentName + "?api-version=" + azureSphereAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"description": "Stackyard Azure Sphere deployment",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateImage",
			method: http.MethodPut,
			path:   catalogBase + "/images/" + imageName + "?api-version=" + azureSphereAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"imageName":   imageName,
					"description": "Stackyard Azure Sphere image",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "UploadImage",
			method: http.MethodPost,
			path:   catalogBase + "/uploadImage?api-version=" + azureSphereAPIVersion,
			payload: map[string]any{
				"name": "uploaded-image",
				"properties": map[string]any{
					"description": "uploaded image",
				},
			},
			statuses: []int{http.StatusAccepted},
		},
		{
			name:     "ListDeviceGroups",
			method:   http.MethodPost,
			path:     catalogBase + "/listDeviceGroups?api-version=" + azureSphereAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDeployments",
			method:   http.MethodPost,
			path:     catalogBase + "/listDeployments?api-version=" + azureSphereAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDeviceInsights",
			method:   http.MethodPost,
			path:     catalogBase + "/listDeviceInsights?api-version=" + azureSphereAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteCatalog",
			method:   http.MethodDelete,
			path:     catalogBase + "?api-version=" + azureSphereAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
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
