package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const extendedLocationAPIVersion = "2021-08-15"
const extendedLocationPrefix = "/azure"

type extendedLocationClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type extendedLocationSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type extendedLocationCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *extendedLocationSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newExtendedLocationClient(endpoint, account, subscriptionKey string) *extendedLocationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"extendedlocation-2021-08-15",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&extendedLocationSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &extendedLocationClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *extendedLocationClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

func buildPath(template string, vars map[string]string) string {
	path := template
	for key, value := range vars {
		path = strings.ReplaceAll(path, "{"+key+"}", url.PathEscape(value))
	}
	return extendedLocationPrefix + path
}

func payloadForMethod(methodPath, location, hostResourceID, clusterExtensionID string) any {
	switch methodPath {
	case "custom-locations/create-or-update":
		return map[string]any{
			"location": location,
			"identity": map[string]any{"type": "SystemAssigned"},
			"properties": map[string]any{
				"hostResourceId":      hostResourceID,
				"namespace":           "custom-location-ns",
				"displayName":         "customLocation01",
				"clusterExtensionIds": []any{clusterExtensionID},
				"authentication": map[string]any{
					"type":  "KubeConfig",
					"value": "<base64 KubeConfig>",
				},
			},
			"tags": map[string]any{"env": "local"},
		}
	case "custom-locations/update":
		return map[string]any{
			"identity": map[string]any{"type": "SystemAssigned"},
			"properties": map[string]any{
				"clusterExtensionIds": []any{clusterExtensionID},
			},
			"tags": map[string]any{"tier": "testing", "archv3": ""},
		}
	}
	return map[string]any{}
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_EXTENDEDLOCATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_EXTENDEDLOCATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-extendedlocation")
	location := getenv("STACKYARD_AZURE_LOCATION", "westus")
	resourceName := getenv("STACKYARD_AZURE_EXTENDEDLOCATION_RESOURCE_NAME", "customLocation01")

	hostResourceID := getenv(
		"STACKYARD_AZURE_EXTENDEDLOCATION_HOST_RESOURCE_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/cluster01", subscriptionID, resourceGroupName),
	)
	clusterExtensionID := getenv(
		"STACKYARD_AZURE_EXTENDEDLOCATION_CLUSTER_EXTENSION_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Kubernetes/connectedClusters/someCluster/providers/Microsoft.KubernetesConfiguration/extensions/fooExtension", subscriptionID, resourceGroupName),
	)

	client := newExtendedLocationClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":    subscriptionID,
		"resourceGroupName": resourceGroupName,
		"resourceName":      resourceName,
	}

	calls := []extendedLocationCall{
		{
			name:     "custom_locations_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ExtendedLocation/customLocations/{resourceName}?api-version=2021-08-15",
			payload:  payloadForMethod("custom-locations/create-or-update", location, hostResourceID, clusterExtensionID),
			statuses: []int{200, 201},
		},
		{
			name:     "custom_locations_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ExtendedLocation/customLocations/{resourceName}?api-version=2021-08-15",
			payload:  nil,
			statuses: []int{202, 204},
		},
		{
			name:     "custom_locations_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ExtendedLocation/customLocations/{resourceName}?api-version=2021-08-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "custom_locations_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ExtendedLocation/customLocations?api-version=2021-08-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "custom_locations_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.ExtendedLocation/customLocations?api-version=2021-08-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "custom_locations_list_enabled_resource_types",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ExtendedLocation/customLocations/{resourceName}/enabledResourceTypes?api-version=2021-08-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "custom_locations_list_operations",
			method:   http.MethodGet,
			template: "/providers/Microsoft.ExtendedLocation/operations?api-version=2021-08-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "custom_locations_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ExtendedLocation/customLocations/{resourceName}?api-version=2021-08-15",
			payload:  payloadForMethod("custom-locations/update", location, hostResourceID, clusterExtensionID),
			statuses: []int{200},
		},
	}

	for _, call := range calls {
		path := buildPath(call.template, vars)
		response, status, err := client.doJSON(ctx, call.method, path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> %d\n", call.name, status)
		if len(response) > 0 {
			encoded, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s marshal response: %v\n", call.name, err)
				os.Exit(1)
			}
			fmt.Println(string(encoded))
		}
	}
}

func ensureStatus(actual int, body []byte, expected ...int) error {
	for _, code := range expected {
		if actual == code {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d, expected one of %v, body: %s", actual, expected, strings.TrimSpace(string(body)))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
