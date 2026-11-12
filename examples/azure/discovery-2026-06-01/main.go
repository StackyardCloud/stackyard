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

const discoveryAPIVersion = "2026-06-01"
const discoveryPrefix = "/azure"

type discoveryClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type discoverySharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type discoveryCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *discoverySharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDiscoveryClient(endpoint, account, subscriptionKey string) *discoveryClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"discovery-2026-06-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&discoverySharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &discoveryClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *discoveryClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return discoveryPrefix + path
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DISCOVERY_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DISCOVERY_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-discovery")
	location := getenv("STACKYARD_AZURE_LOCATION", "uksouth")

	workspaceName := getenv("STACKYARD_AZURE_DISCOVERY_WORKSPACE_NAME", "workspace01")
	bookshelfName := getenv("STACKYARD_AZURE_DISCOVERY_BOOKSHELF_NAME", "bookshelf01")
	storageContainerName := getenv("STACKYARD_AZURE_DISCOVERY_STORAGE_CONTAINER_NAME", "storagecontainer01")
	storageAssetName := getenv("STACKYARD_AZURE_DISCOVERY_STORAGE_ASSET_NAME", "storageasset01")
	supercomputerName := getenv("STACKYARD_AZURE_DISCOVERY_SUPERCOMPUTER_NAME", "supercomputer01")
	nodePoolName := getenv("STACKYARD_AZURE_DISCOVERY_NODE_POOL_NAME", "nodepool01")
	projectName := getenv("STACKYARD_AZURE_DISCOVERY_PROJECT_NAME", "project01")
	chatModelDeploymentName := getenv("STACKYARD_AZURE_DISCOVERY_CHAT_MODEL_DEPLOYMENT_NAME", "chatdeploy01")
	toolName := getenv("STACKYARD_AZURE_DISCOVERY_TOOL_NAME", "tool01")
	privateEndpointConnectionName := getenv("STACKYARD_AZURE_DISCOVERY_PRIVATE_ENDPOINT_CONNECTION_NAME", "connection")
	privateLinkResourceName := getenv("STACKYARD_AZURE_DISCOVERY_PRIVATE_LINK_RESOURCE_NAME", "plr1")

	client := newDiscoveryClient(endpoint, account, subscriptionKey)

	workspaceManagedIdentityID := getenv(
		"STACKYARD_AZURE_DISCOVERY_WORKSPACE_MANAGED_IDENTITY_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/managedid1", subscriptionID, resourceGroupName),
	)
	privateEndpointID := getenv(
		"STACKYARD_AZURE_DISCOVERY_PRIVATE_ENDPOINT_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/privateEndpoints/privateEndpoint1", subscriptionID, resourceGroupName),
	)
	supercomputerID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Discovery/supercomputers/%s",
		subscriptionID,
		resourceGroupName,
		supercomputerName,
	)
	storageAccountContainerPath := getenv("STACKYARD_AZURE_DISCOVERY_STORAGE_CONTAINER_PATH", "https://examplestorage.blob.core.windows.net/discovery")

	vars := map[string]string{
		"subscriptionId":                subscriptionID,
		"resourceGroupName":             resourceGroupName,
		"workspaceName":                 workspaceName,
		"bookshelfName":                 bookshelfName,
		"storageContainerName":          storageContainerName,
		"storageAssetName":              storageAssetName,
		"supercomputerName":             supercomputerName,
		"nodePoolName":                  nodePoolName,
		"projectName":                   projectName,
		"chatModelDeploymentName":       chatModelDeploymentName,
		"toolName":                      toolName,
		"privateEndpointConnectionName": privateEndpointConnectionName,
		"privateLinkResourceName":       privateLinkResourceName,
	}

	payloadBookshelfPrivateEndpointConnectionCreate := map[string]any{
		"properties": map[string]any{
			"privateEndpoint": map[string]any{
				"id": privateEndpointID,
			},
			"privateLinkServiceConnectionState": map[string]any{
				"status":          "Pending",
				"description":     "awaiting approval",
				"actionsRequired": "None",
			},
		},
	}

	payloadBookshelfCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"env": "local", "service": "discovery"},
		"properties": map[string]any{
			"description": "Example bookshelf",
		},
	}

	payloadBookshelfUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
		"properties": map[string]any{
			"description": "Updated example bookshelf",
		},
	}

	payloadChatModelDeploymentCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"env": "local"},
		"properties": map[string]any{
			"modelFormat":  "openai",
			"modelName":    "gpt-4o-mini",
			"modelVersion": "2026-01-01",
			"skuName":      "standard",
			"capacity":     1,
		},
	}

	payloadChatModelDeploymentUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
		"properties": map[string]any{
			"capacity": 2,
		},
	}

	payloadNodePoolCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"env": "local"},
		"properties": map[string]any{
			"description": "Primary node pool",
		},
	}

	payloadNodePoolUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
		"properties": map[string]any{
			"description": "Updated node pool",
		},
	}

	payloadProjectCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"env": "local"},
		"properties": map[string]any{
			"description": "Example project",
		},
	}

	payloadProjectUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
		"properties": map[string]any{
			"description": "Updated project",
		},
	}

	payloadStorageAssetCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"env": "local"},
		"properties": map[string]any{
			"description": "Example storage asset",
			"path":        storageAccountContainerPath + "/asset-a",
		},
	}

	payloadStorageAssetUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
		"properties": map[string]any{
			"description": "Updated storage asset",
			"path":        storageAccountContainerPath + "/asset-b",
		},
	}

	payloadStorageContainerCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"env": "local"},
		"properties": map[string]any{
			"path": storageAccountContainerPath,
		},
	}

	payloadStorageContainerUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
		"properties": map[string]any{
			"path": storageAccountContainerPath + "/updated",
		},
	}

	payloadSupercomputerCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"env": "local"},
		"properties": map[string]any{
			"description": "Example supercomputer",
		},
	}

	payloadSupercomputerUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
		"properties": map[string]any{
			"description": "Updated supercomputer",
		},
	}

	payloadToolCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"env": "local"},
		"properties": map[string]any{
			"description": "Example tool",
		},
	}

	payloadToolUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
		"properties": map[string]any{
			"description": "Updated tool",
		},
	}

	payloadWorkspacePrivateEndpointConnectionCreate := map[string]any{
		"properties": map[string]any{
			"privateEndpoint": map[string]any{
				"id": privateEndpointID,
			},
			"privateLinkServiceConnectionState": map[string]any{
				"status":          "Pending",
				"description":     "awaiting approval",
				"actionsRequired": "None",
			},
		},
	}

	payloadWorkspaceCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"env": "local"},
		"properties": map[string]any{
			"supercomputerIds": []any{supercomputerID},
			"workspaceIdentity": map[string]any{
				"id": workspaceManagedIdentityID,
			},
			"publicNetworkAccess": "Enabled",
		},
	}

	payloadWorkspaceUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
		"properties": map[string]any{
			"publicNetworkAccess": "Enabled",
		},
	}

	calls := []discoveryCall{
		{
			name:     "operations_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Discovery/operations?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "bookshelves_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}?api-version=2026-06-01",
			payload:  payloadBookshelfCreate,
			statuses: []int{200, 201},
		}, {
			name:     "storage_containers_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}?api-version=2026-06-01",
			payload:  payloadStorageContainerCreate,
			statuses: []int{200, 201},
		}, {
			name:     "supercomputers_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}?api-version=2026-06-01",
			payload:  payloadSupercomputerCreate,
			statuses: []int{200, 201},
		}, {
			name:     "workspaces_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}?api-version=2026-06-01",
			payload:  payloadWorkspaceCreate,
			statuses: []int{200, 201},
		}, {
			name:     "bookshelves_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "storage_containers_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "supercomputers_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "workspaces_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "bookshelves_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "bookshelves_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Discovery/bookshelves?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "storage_containers_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "storage_containers_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Discovery/storageContainers?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "supercomputers_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "supercomputers_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Discovery/supercomputers?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "workspaces_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "workspaces_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Discovery/workspaces?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "bookshelves_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}?api-version=2026-06-01",
			payload:  payloadBookshelfUpdate,
			statuses: []int{200, 202},
		}, {
			name:     "storage_containers_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}?api-version=2026-06-01",
			payload:  payloadStorageContainerUpdate,
			statuses: []int{200, 202},
		}, {
			name:     "supercomputers_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}?api-version=2026-06-01",
			payload:  payloadSupercomputerUpdate,
			statuses: []int{200, 202},
		}, {
			name:     "workspaces_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}?api-version=2026-06-01",
			payload:  payloadWorkspaceUpdate,
			statuses: []int{200, 202},
		}, {
			name:     "bookshelves_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{202, 204},
		}, {
			name:     "storage_containers_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{202, 204},
		}, {
			name:     "supercomputers_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{202, 204},
		}, {
			name:     "workspaces_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{202, 204},
		}, {
			name:     "chat_model_deployments_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/chatModelDeployments/{chatModelDeploymentName}?api-version=2026-06-01",
			payload:  payloadChatModelDeploymentCreate,
			statuses: []int{200, 201},
		}, {
			name:     "node_pools_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}/nodePools/{nodePoolName}?api-version=2026-06-01",
			payload:  payloadNodePoolCreate,
			statuses: []int{200, 201},
		}, {
			name:     "projects_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/projects/{projectName}?api-version=2026-06-01",
			payload:  payloadProjectCreate,
			statuses: []int{200, 201},
		}, {
			name:     "storage_assets_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}/storageAssets/{storageAssetName}?api-version=2026-06-01",
			payload:  payloadStorageAssetCreate,
			statuses: []int{200, 201},
		}, {
			name:     "tools_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/tools/{toolName}?api-version=2026-06-01",
			payload:  payloadToolCreate,
			statuses: []int{200, 201},
		}, {
			name:     "chat_model_deployments_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/chatModelDeployments/{chatModelDeploymentName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "node_pools_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}/nodePools/{nodePoolName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "projects_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/projects/{projectName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "storage_assets_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}/storageAssets/{storageAssetName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "tools_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/tools/{toolName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "chat_model_deployments_list_by_workspace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/chatModelDeployments?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "node_pools_list_by_supercomputer",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}/nodePools?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "projects_list_by_workspace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/projects?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "storage_assets_list_by_storage_container",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}/storageAssets?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "tools_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/tools?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "tools_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Discovery/tools?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "chat_model_deployments_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/chatModelDeployments/{chatModelDeploymentName}?api-version=2026-06-01",
			payload:  payloadChatModelDeploymentUpdate,
			statuses: []int{200, 202},
		}, {
			name:     "node_pools_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}/nodePools/{nodePoolName}?api-version=2026-06-01",
			payload:  payloadNodePoolUpdate,
			statuses: []int{200, 202},
		}, {
			name:     "projects_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/projects/{projectName}?api-version=2026-06-01",
			payload:  payloadProjectUpdate,
			statuses: []int{200, 202},
		}, {
			name:     "storage_assets_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}/storageAssets/{storageAssetName}?api-version=2026-06-01",
			payload:  payloadStorageAssetUpdate,
			statuses: []int{200, 202},
		}, {
			name:     "tools_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/tools/{toolName}?api-version=2026-06-01",
			payload:  payloadToolUpdate,
			statuses: []int{200, 202},
		}, {
			name:     "chat_model_deployments_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/chatModelDeployments/{chatModelDeploymentName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{202, 204},
		}, {
			name:     "node_pools_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}/nodePools/{nodePoolName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{202, 204},
		}, {
			name:     "projects_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/projects/{projectName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{202, 204},
		}, {
			name:     "storage_assets_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}/storageAssets/{storageAssetName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{202, 204},
		}, {
			name:     "tools_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/tools/{toolName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{202, 204},
		}, {
			name:     "bookshelf_private_endpoint_connections_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-06-01",
			payload:  payloadBookshelfPrivateEndpointConnectionCreate,
			statuses: []int{200, 201},
		}, {
			name:     "workspace_private_endpoint_connections_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-06-01",
			payload:  payloadWorkspacePrivateEndpointConnectionCreate,
			statuses: []int{200, 201},
		}, {
			name:     "bookshelf_private_endpoint_connections_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "workspace_private_endpoint_connections_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "bookshelf_private_endpoint_connections_list_by_bookshelf",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}/privateEndpointConnections?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "workspace_private_endpoint_connections_list_by_workspace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/privateEndpointConnections?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "bookshelf_private_endpoint_connections_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{202, 204},
		}, {
			name:     "workspace_private_endpoint_connections_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{202, 204},
		}, {
			name:     "bookshelf_private_link_resources_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}/privateLinkResources/{privateLinkResourceName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "workspace_private_link_resources_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/privateLinkResources/{privateLinkResourceName}?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "bookshelf_private_link_resources_list_by_bookshelf",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}/privateLinkResources?api-version=2026-06-01",
			payload:  nil,
			statuses: []int{200},
		}, {
			name:     "workspace_private_link_resources_list_by_workspace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/privateLinkResources?api-version=2026-06-01",
			payload:  nil,
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
