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

const elasticSanAPIVersion = "2025-09-01"
const elasticSanPrefix = "/azure"

type elasticSanClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type elasticSanSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type elasticSanCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *elasticSanSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newElasticSanClient(endpoint, account, subscriptionKey string) *elasticSanClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"elasticsan-2025-09-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&elasticSanSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &elasticSanClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *elasticSanClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return elasticSanPrefix + path
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_ELASTICSAN_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_ELASTICSAN_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-elasticsan")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	elasticSanName := getenv("STACKYARD_AZURE_ELASTICSAN_NAME", "esanexample01")
	volumeGroupName := getenv("STACKYARD_AZURE_ELASTICSAN_VOLUME_GROUP_NAME", "volumegroup01")
	volumeName := getenv("STACKYARD_AZURE_ELASTICSAN_VOLUME_NAME", "volume01")
	snapshotName := getenv("STACKYARD_AZURE_ELASTICSAN_SNAPSHOT_NAME", "snapshot01")
	privateEndpointConnectionName := getenv("STACKYARD_AZURE_ELASTICSAN_PRIVATE_ENDPOINT_CONNECTION_NAME", "privateendpointconnection01")
	privateEndpointName := getenv("STACKYARD_AZURE_ELASTICSAN_PRIVATE_ENDPOINT_NAME", "private-endpoint-01")
	virtualNetworkName := getenv("STACKYARD_AZURE_ELASTICSAN_VNET_NAME", "esan-vnet")
	subnetName := getenv("STACKYARD_AZURE_ELASTICSAN_SUBNET_NAME", "esan-subnet")
	diskSnapshotName := getenv("STACKYARD_AZURE_ELASTICSAN_DISK_SNAPSHOT_NAME", "disk-snapshot-01")
	managedByVMName := getenv("STACKYARD_AZURE_ELASTICSAN_MANAGED_BY_VM_NAME", "esan-vm")

	client := newElasticSanClient(endpoint, account, subscriptionKey)

	privateEndpointID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/privateEndpoints/%s",
		subscriptionID,
		resourceGroupName,
		privateEndpointName,
	)
	virtualNetworkRuleID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
		subscriptionID,
		resourceGroupName,
		virtualNetworkName,
		subnetName,
	)
	volumeResourceID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ElasticSan/elasticSans/%s/volumegroups/%s/volumes/%s",
		subscriptionID,
		resourceGroupName,
		elasticSanName,
		volumeGroupName,
		volumeName,
	)
	diskSnapshotID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/snapshots/%s",
		subscriptionID,
		resourceGroupName,
		diskSnapshotName,
	)
	managedByResourceID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s",
		subscriptionID,
		resourceGroupName,
		managedByVMName,
	)

	vars := map[string]string{
		"subscriptionId":                subscriptionID,
		"resourceGroupName":             resourceGroupName,
		"elasticSanName":                elasticSanName,
		"privateEndpointConnectionName": privateEndpointConnectionName,
		"volumeGroupName":               volumeGroupName,
		"volumeName":                    volumeName,
		"snapshotName":                  snapshotName,
	}

	elasticSanCreatePayload := map[string]any{
		"location": location,
		"tags": map[string]any{
			"environment": "local",
			"service":     "elasticsan",
		},
		"properties": map[string]any{
			"baseSizeTiB":             5,
			"extendedCapacitySizeTiB": 25,
			"availabilityZones":       []any{"1"},
			"publicNetworkAccess":     "Enabled",
			"sku": map[string]any{
				"name": "Premium_LRS",
				"tier": "Premium",
			},
			"autoScaleProperties": map[string]any{
				"scaleUpProperties": map[string]any{
					"autoScalePolicyEnforcement":  "None",
					"capacityUnitScaleUpLimitTiB": 17,
					"increaseCapacityUnitByTiB":   4,
					"unusedSizeTiB":               24,
				},
			},
		},
	}

	elasticSanUpdatePayload := map[string]any{
		"properties": map[string]any{
			"baseSizeTiB":             13,
			"extendedCapacitySizeTiB": 29,
			"publicNetworkAccess":     "Enabled",
			"autoScaleProperties": map[string]any{
				"scaleUpProperties": map[string]any{
					"autoScalePolicyEnforcement":  "None",
					"capacityUnitScaleUpLimitTiB": 17,
					"increaseCapacityUnitByTiB":   4,
					"unusedSizeTiB":               24,
				},
			},
		},
		"tags": map[string]any{
			"environment": "local-updated",
			"service":     "elasticsan",
		},
	}

	privateEndpointConnectionCreatePayload := map[string]any{
		"properties": map[string]any{
			"privateLinkServiceConnectionState": map[string]any{
				"status":          "Approved",
				"description":     "Auto-approved in local emulation",
				"actionsRequired": "None",
			},
			"groupIds": []any{"elasticSan"},
			"privateEndpoint": map[string]any{
				"id": privateEndpointID,
			},
		},
	}

	volumeGroupCreatePayload := map[string]any{
		"identity": map[string]any{
			"type": "None",
		},
		"properties": map[string]any{
			"encryption":                        "EncryptionAtRestWithPlatformKey",
			"enforceDataIntegrityCheckForIscsi": true,
			"protocolType":                      "Iscsi",
			"networkAcls": map[string]any{
				"virtualNetworkRules": []any{
					map[string]any{
						"id":     virtualNetworkRuleID,
						"action": "Allow",
					},
				},
			},
		},
	}

	volumeGroupUpdatePayload := map[string]any{
		"identity": map[string]any{
			"type": "None",
		},
		"properties": map[string]any{
			"encryption":                        "EncryptionAtRestWithPlatformKey",
			"enforceDataIntegrityCheckForIscsi": true,
			"protocolType":                      "Iscsi",
		},
	}

	volumeSnapshotCreatePayload := map[string]any{
		"properties": map[string]any{
			"creationData": map[string]any{
				"sourceId": volumeResourceID,
			},
		},
	}

	volumeCreatePayload := map[string]any{
		"properties": map[string]any{
			"sizeGiB": 23,
			"creationData": map[string]any{
				"createSource": "None",
				"sourceId":     volumeResourceID,
			},
			"managedBy": map[string]any{
				"resourceId": managedByResourceID,
			},
		},
	}

	volumeUpdatePayload := map[string]any{
		"properties": map[string]any{
			"sizeGiB": 7,
			"managedBy": map[string]any{
				"resourceId": managedByResourceID,
			},
		},
	}

	preBackupPayload := map[string]any{
		"volumeNames": []any{volumeName},
	}

	preRestorePayload := map[string]any{
		"diskSnapshotIds": []any{diskSnapshotID},
	}

	calls := []elasticSanCall{
		{
			name:     "operations_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.ElasticSan/operations?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "elastic_sans_create",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}?api-version=2025-09-01",
			payload:  elasticSanCreatePayload,
			statuses: []int{200, 201},
		},
		{
			name:     "elastic_sans_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "elastic_sans_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "elastic_sans_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.ElasticSan/elasticSans?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "elastic_sans_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}?api-version=2025-09-01",
			payload:  elasticSanUpdatePayload,
			statuses: []int{200, 202},
		},
		{
			name:     "private_endpoint_connections_create",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2025-09-01",
			payload:  privateEndpointConnectionCreatePayload,
			statuses: []int{200, 201},
		},
		{
			name:     "private_endpoint_connections_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "private_endpoint_connections_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/privateEndpointConnections?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "private_link_resources_list_by_elastic_san",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/privateLinkResources?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "skus_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.ElasticSan/skus?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "volume_groups_create",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}?api-version=2025-09-01",
			payload:  volumeGroupCreatePayload,
			statuses: []int{200, 201},
		},
		{
			name:     "volume_groups_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "volume_groups_list_by_elastic_san",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "volume_groups_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}?api-version=2025-09-01",
			payload:  volumeGroupUpdatePayload,
			statuses: []int{200, 202},
		},
		{
			name:     "volume_snapshots_create",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/snapshots/{snapshotName}?api-version=2025-09-01",
			payload:  volumeSnapshotCreatePayload,
			statuses: []int{200, 201},
		},
		{
			name:     "volume_snapshots_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/snapshots/{snapshotName}?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "volume_snapshots_list_by_volume_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/snapshots?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "volumes_create",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/volumes/{volumeName}?api-version=2025-09-01",
			payload:  volumeCreatePayload,
			statuses: []int{200, 201},
		},
		{
			name:     "volumes_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/volumes/{volumeName}?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "volumes_list_by_volume_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/volumes?api-version=2025-09-01",
			statuses: []int{200},
		},
		{
			name:     "volumes_pre_backup",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/preBackup?api-version=2025-09-01",
			payload:  preBackupPayload,
			statuses: []int{200, 202},
		},
		{
			name:     "volumes_pre_restore",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/preRestore?api-version=2025-09-01",
			payload:  preRestorePayload,
			statuses: []int{200, 202},
		},
		{
			name:     "volumes_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/volumes/{volumeName}?api-version=2025-09-01",
			payload:  volumeUpdatePayload,
			statuses: []int{200, 202},
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

	cleanup := []elasticSanCall{
		{
			name:     "volumes_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/volumes/{volumeName}?api-version=2025-09-01",
			statuses: []int{200, 202, 204},
		},
		{
			name:     "volume_snapshots_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/snapshots/{snapshotName}?api-version=2025-09-01",
			statuses: []int{200, 202, 204},
		},
		{
			name:     "volume_groups_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}?api-version=2025-09-01",
			statuses: []int{200, 202, 204},
		},
		{
			name:     "private_endpoint_connections_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2025-09-01",
			statuses: []int{200, 202, 204},
		},
		{
			name:     "elastic_sans_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}?api-version=2025-09-01",
			statuses: []int{200, 202, 204},
		},
	}

	for _, call := range cleanup {
		path := buildPath(call.template, vars)
		response, status, err := client.doJSON(ctx, call.method, path, nil, call.statuses...)
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
