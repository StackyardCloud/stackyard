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

const deviceregistryAPIVersion = "2026-04-01"
const deviceregistryPrefix = "/azure"

type deviceregistryClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type deviceregistrySharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type deviceregistryCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *deviceregistrySharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDeviceregistryClient(endpoint, account, subscriptionKey string) *deviceregistryClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"deviceregistry-2026-04-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&deviceregistrySharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &deviceregistryClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *deviceregistryClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return deviceregistryPrefix + path + "?api-version=" + deviceregistryAPIVersion
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DEVICEREGISTRY_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DEVICEREGISTRY_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "myResourceGroup")
	location := getenv("STACKYARD_AZURE_LOCATION", "westeurope")

	assetEndpointProfileName := getenv("STACKYARD_AZURE_DEVICEREGISTRY_AEP_NAME", "my-assetendpointprofile")
	assetName := getenv("STACKYARD_AZURE_DEVICEREGISTRY_ASSET_NAME", "my-asset")
	billingContainerName := getenv("STACKYARD_AZURE_DEVICEREGISTRY_BILLING_CONTAINER_NAME", "my-billing-container")
	namespaceName := getenv("STACKYARD_AZURE_DEVICEREGISTRY_NAMESPACE_NAME", "my-namespace-1")
	deviceName := getenv("STACKYARD_AZURE_DEVICEREGISTRY_DEVICE_NAME", "device1")
	discoveredAssetName := getenv("STACKYARD_AZURE_DEVICEREGISTRY_DISCOVERED_ASSET_NAME", "my-discoveredasset-1")
	discoveredDeviceName := getenv("STACKYARD_AZURE_DEVICEREGISTRY_DISCOVERED_DEVICE_NAME", "my-discovereddevice-1")
	schemaRegistryName := getenv("STACKYARD_AZURE_DEVICEREGISTRY_SCHEMA_REGISTRY_NAME", "my-schema-registry")
	schemaName := getenv("STACKYARD_AZURE_DEVICEREGISTRY_SCHEMA_NAME", "my-schema")
	schemaVersionName := getenv("STACKYARD_AZURE_DEVICEREGISTRY_SCHEMA_VERSION_NAME", "v1")
	operationID := getenv("STACKYARD_AZURE_DEVICEREGISTRY_OPERATION_ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")

	customLocationID := getenv(
		"STACKYARD_AZURE_DEVICEREGISTRY_CUSTOM_LOCATION_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ExtendedLocation/customLocations/location1",
			subscriptionID,
			resourceGroupName,
		),
	)
	storageAccountContainerURL := getenv("STACKYARD_AZURE_DEVICEREGISTRY_SCHEMA_STORAGE_CONTAINER_URL", "my-blob-storage.blob.core.windows.net/my-container")

	client := newDeviceregistryClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":           subscriptionID,
		"resourceGroupName":        resourceGroupName,
		"assetEndpointProfileName": assetEndpointProfileName,
		"assetName":                assetName,
		"billingContainerName":     billingContainerName,
		"namespaceName":            namespaceName,
		"deviceName":               deviceName,
		"discoveredAssetName":      discoveredAssetName,
		"discoveredDeviceName":     discoveredDeviceName,
		"schemaRegistryName":       schemaRegistryName,
		"schemaName":               schemaName,
		"schemaVersionName":        schemaVersionName,
		"location":                 location,
		"operationId":              operationID,
	}

	payloadAssetEndpointProfileCreate := map[string]any{
		"location": location,
		"extendedLocation": map[string]any{
			"type": "CustomLocation",
			"name": customLocationID,
		},
		"tags": map[string]any{"site": "building-1"},
		"properties": map[string]any{
			"endpointProfileType": "myEndpointProfileType",
			"targetAddress":       "https://www.example.com/myTargetAddress",
			"authentication": map[string]any{
				"method": "Anonymous",
			},
		},
	}

	payloadAssetEndpointProfileUpdate := map[string]any{
		"tags": map[string]any{"site": "building-2"},
		"properties": map[string]any{
			"targetAddress": "https://www.example.com/myUpdatedTargetAddress",
		},
	}

	payloadAssetCreate := map[string]any{
		"location": location,
		"extendedLocation": map[string]any{
			"type": "CustomLocation",
			"name": customLocationID,
		},
		"tags": map[string]any{"site": "building-1"},
		"properties": map[string]any{
			"assetEndpointProfileRef": assetEndpointProfileName,
			"enabled":                 true,
			"displayName":             "AssetDisplayName",
			"description":             "This is a sample Asset",
		},
	}

	payloadAssetUpdate := map[string]any{
		"tags": map[string]any{"site": "building-2"},
		"properties": map[string]any{
			"enabled":     true,
			"description": "Updated sample Asset",
		},
	}

	payloadNamespaceAssetCreate := map[string]any{
		"location": location,
		"extendedLocation": map[string]any{
			"type": "CustomLocation",
			"name": customLocationID,
		},
		"tags": map[string]any{"site": "building-1"},
		"properties": map[string]any{
			"deviceRef": map[string]any{
				"deviceName":   deviceName,
				"endpointName": "opcuaendpointname",
			},
			"enabled":     true,
			"displayName": "NamespaceAssetDisplayName",
		},
	}

	payloadNamespaceAssetUpdate := map[string]any{
		"tags": map[string]any{"site": "building-2"},
		"properties": map[string]any{
			"description": "Updated namespace asset",
		},
	}

	payloadNamespaceAssetExecuteAction := map[string]any{
		"managementActionName": "action1",
		"managementGroupName":  "managementGroup1",
		"payload":              map[string]any{"arg1": "value1"},
	}

	payloadNamespaceDeviceCreate := map[string]any{
		"location": location,
		"extendedLocation": map[string]any{
			"type": "CustomLocation",
			"name": customLocationID,
		},
		"tags": map[string]any{"site": "building-1"},
		"properties": map[string]any{
			"enabled":          true,
			"externalDeviceId": "external-device-1",
			"manufacturer":     "Contoso",
			"model":            "ContosoDeviceModel",
		},
	}

	payloadNamespaceDeviceUpdate := map[string]any{
		"tags": map[string]any{"site": "building-2"},
		"properties": map[string]any{
			"operatingSystemVersion": "1.0.1",
		},
	}

	payloadNamespaceDiscoveredAssetCreate := map[string]any{
		"location": location,
		"extendedLocation": map[string]any{
			"type": "CustomLocation",
			"name": customLocationID,
		},
		"tags": map[string]any{"site": "building-1"},
		"properties": map[string]any{
			"deviceRef": map[string]any{
				"deviceName":   deviceName,
				"endpointName": "opcuaendpointname",
			},
			"discoveryId": "discovery-1",
			"version":     1,
			"displayName": "NamespaceDiscoveredAssetDisplayName",
		},
	}

	payloadNamespaceDiscoveredAssetUpdate := map[string]any{
		"tags": map[string]any{"site": "building-2"},
		"properties": map[string]any{
			"discoveryId": "discovery-1",
			"version":     2,
			"description": "Updated discovered asset",
			"deviceRef": map[string]any{
				"deviceName":   deviceName,
				"endpointName": "opcuaendpointname",
			},
		},
	}

	payloadNamespaceDiscoveredDeviceCreate := map[string]any{
		"location": location,
		"extendedLocation": map[string]any{
			"type": "CustomLocation",
			"name": customLocationID,
		},
		"tags": map[string]any{"site": "building-1"},
		"properties": map[string]any{
			"discoveryId":      "discovery-1",
			"version":          1,
			"externalDeviceId": "external-device-1",
			"manufacturer":     "Contoso",
			"model":            "ContosoDeviceModel",
		},
	}

	payloadNamespaceDiscoveredDeviceUpdate := map[string]any{
		"tags": map[string]any{"site": "building-2"},
		"properties": map[string]any{
			"discoveryId": "discovery-1",
			"version":     2,
		},
	}

	payloadNamespaceCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"site": "building-1"},
		"properties": map[string]any{
			"messaging": map[string]any{
				"mqttBroker": map[string]any{
					"resourceId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-rg/providers/Microsoft.EventGrid/namespaces/test-ns",
				},
			},
		},
	}

	payloadNamespaceUpdate := map[string]any{
		"tags": map[string]any{"site": "building-2"},
	}

	payloadNamespaceMigrate := map[string]any{
		"resourceIds": []any{
			fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DeviceRegistry/assets/%s", subscriptionID, resourceGroupName, assetName),
		},
		"scope": "Resources",
	}

	payloadSchemaRegistryCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"site": "building-1"},
		"properties": map[string]any{
			"namespace":                  "sr-namespace-001",
			"displayName":                "Schema Registry namespace 001",
			"description":                "This is a sample Schema Registry",
			"storageAccountContainerUrl": storageAccountContainerURL,
		},
	}

	payloadSchemaRegistryUpdate := map[string]any{
		"tags": map[string]any{"site": "building-2"},
		"properties": map[string]any{
			"description": "Updated schema registry",
		},
	}

	payloadSchemaCreate := map[string]any{
		"properties": map[string]any{
			"format":      "Json",
			"schemaType":  "Message",
			"displayName": "My Schema",
			"description": "Schema used by namespace assets",
		},
	}

	payloadSchemaVersionCreate := map[string]any{
		"properties": map[string]any{
			"schemaContent": `{"$schema":"http://json-schema.org/draft-07/schema#","title":"Telemetry","type":"object","properties":{"temperature":{"type":"number"}}}`,
			"description":   "Initial schema version",
		},
	}

	calls := []deviceregistryCall{
		{
			name:     "AssetEndpointProfilesCreateOrReplace",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assetEndpointProfiles/{assetEndpointProfileName}",
			payload:  payloadAssetEndpointProfileCreate,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "AssetEndpointProfilesDelete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assetEndpointProfiles/{assetEndpointProfileName}",
			payload:  nil,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "AssetEndpointProfilesGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assetEndpointProfiles/{assetEndpointProfileName}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AssetEndpointProfilesListByResourceGroup",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assetEndpointProfiles",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AssetEndpointProfilesListBySubscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.DeviceRegistry/assetEndpointProfiles",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AssetEndpointProfilesUpdate",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assetEndpointProfiles/{assetEndpointProfileName}",
			payload:  payloadAssetEndpointProfileUpdate,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "AssetsCreateOrReplace",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assets/{assetName}",
			payload:  payloadAssetCreate,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "AssetsDelete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assets/{assetName}",
			payload:  nil,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "AssetsGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assets/{assetName}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AssetsListByResourceGroup",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assets",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AssetsListBySubscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.DeviceRegistry/assets",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AssetsUpdate",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assets/{assetName}",
			payload:  payloadAssetUpdate,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "BillingContainersGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.DeviceRegistry/billingContainers/{billingContainerName}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "BillingContainersListBySubscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.DeviceRegistry/billingContainers",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespaceAssetsCreateOrReplace",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/assets/{assetName}",
			payload:  payloadNamespaceAssetCreate,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "NamespaceAssetsDelete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/assets/{assetName}",
			payload:  nil,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "NamespaceAssetsExecuteAction",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/assets/{assetName}/executeAction",
			payload:  payloadNamespaceAssetExecuteAction,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "NamespaceAssetsGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/assets/{assetName}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespaceAssetsListByResourceGroup",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/assets",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespaceAssetsUpdate",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/assets/{assetName}",
			payload:  payloadNamespaceAssetUpdate,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "NamespaceDevicesCreateOrReplace",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/devices/{deviceName}",
			payload:  payloadNamespaceDeviceCreate,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "NamespaceDevicesDelete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/devices/{deviceName}",
			payload:  nil,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "NamespaceDevicesGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/devices/{deviceName}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespaceDevicesListByResourceGroup",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/devices",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespaceDevicesUpdate",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/devices/{deviceName}",
			payload:  payloadNamespaceDeviceUpdate,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "NamespaceDiscoveredAssetsCreateOrReplace",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredAssets/{discoveredAssetName}",
			payload:  payloadNamespaceDiscoveredAssetCreate,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "NamespaceDiscoveredAssetsDelete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredAssets/{discoveredAssetName}",
			payload:  nil,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "NamespaceDiscoveredAssetsGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredAssets/{discoveredAssetName}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespaceDiscoveredAssetsListByResourceGroup",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredAssets",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespaceDiscoveredAssetsUpdate",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredAssets/{discoveredAssetName}",
			payload:  payloadNamespaceDiscoveredAssetUpdate,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "NamespaceDiscoveredDevicesCreateOrReplace",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredDevices/{discoveredDeviceName}",
			payload:  payloadNamespaceDiscoveredDeviceCreate,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "NamespaceDiscoveredDevicesDelete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredDevices/{discoveredDeviceName}",
			payload:  nil,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "NamespaceDiscoveredDevicesGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredDevices/{discoveredDeviceName}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespaceDiscoveredDevicesListByResourceGroup",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredDevices",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespaceDiscoveredDevicesUpdate",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredDevices/{discoveredDeviceName}",
			payload:  payloadNamespaceDiscoveredDeviceUpdate,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "NamespacesCreateOrReplace",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}",
			payload:  payloadNamespaceCreate,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "NamespacesDelete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}",
			payload:  nil,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "NamespacesGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespacesListByResourceGroup",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespacesListBySubscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.DeviceRegistry/namespaces",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NamespacesMigrate",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/migrate",
			payload:  payloadNamespaceMigrate,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "NamespacesUpdate",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}",
			payload:  payloadNamespaceUpdate,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "OperationStatusGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.DeviceRegistry/locations/{location}/operationStatuses/{operationId}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "OperationsList",
			method:   http.MethodGet,
			template: "/providers/Microsoft.DeviceRegistry/operations",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "SchemaRegistriesCreateOrReplace",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}",
			payload:  payloadSchemaRegistryCreate,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "SchemaRegistriesDelete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}",
			payload:  nil,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "SchemaRegistriesGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "SchemaRegistriesListByResourceGroup",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "SchemaRegistriesListBySubscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.DeviceRegistry/schemaRegistries",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "SchemaRegistriesUpdate",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}",
			payload:  payloadSchemaRegistryUpdate,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "SchemaVersionsCreateOrReplace",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas/{schemaName}/schemaVersions/{schemaVersionName}",
			payload:  payloadSchemaVersionCreate,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "SchemaVersionsDelete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas/{schemaName}/schemaVersions/{schemaVersionName}",
			payload:  nil,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "SchemaVersionsGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas/{schemaName}/schemaVersions/{schemaVersionName}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "SchemaVersionsListBySchema",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas/{schemaName}/schemaVersions",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "SchemasCreateOrReplace",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas/{schemaName}",
			payload:  payloadSchemaCreate,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "SchemasDelete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas/{schemaName}",
			payload:  nil,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "SchemasGet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas/{schemaName}",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "SchemasListBySchemaRegistry",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas",
			payload:  nil,
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		path := buildPath(call.template, vars)
		response, status, err := client.doJSON(ctx, call.method, path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, response)
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
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("%s -> %d\n%s\n", name, status, string(formatted))
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
