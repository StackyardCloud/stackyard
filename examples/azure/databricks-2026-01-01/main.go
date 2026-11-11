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

const databricksAPIVersion = "2026-01-01"

type databricksClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type databricksSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *databricksSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDatabricksClient(endpoint, account, subscriptionKey string) *databricksClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"databricks-2026-01-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&databricksSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &databricksClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *databricksClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DATABRICKS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DATABRICKS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-databricks")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus2")
	workspaceName := getenv("STACKYARD_AZURE_DATABRICKS_WORKSPACE_NAME", "databricks-workspace-example")
	connectorName := getenv("STACKYARD_AZURE_DATABRICKS_ACCESS_CONNECTOR_NAME", "databricks-access-connector-example")
	peeringName := getenv("STACKYARD_AZURE_DATABRICKS_VNET_PEERING_NAME", "databricks-vnet-peering-example")
	privateEndpointConnectionName := getenv("STACKYARD_AZURE_DATABRICKS_PRIVATE_ENDPOINT_CONNECTION_NAME", "databricks-workspace-example.23456789-1111-1111-1111-111111111111")
	privateLinkResourceName := getenv("STACKYARD_AZURE_DATABRICKS_PRIVATE_LINK_RESOURCE_NAME", "databricks_ui_api")

	managedResourceGroupID := getenv(
		"STACKYARD_AZURE_DATABRICKS_MANAGED_RESOURCE_GROUP_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s-managed",
			subscriptionID,
			resourceGroup,
		),
	)
	accessConnectorResourceID := getenv(
		"STACKYARD_AZURE_DATABRICKS_ACCESS_CONNECTOR_RESOURCE_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Databricks/accessConnectors/%s",
			subscriptionID,
			resourceGroup,
			connectorName,
		),
	)
	remoteVirtualNetworkID := getenv(
		"STACKYARD_AZURE_DATABRICKS_REMOTE_VNET_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/databricks-remote-vnet",
			subscriptionID,
			resourceGroup,
		),
	)
	privateEndpointID := getenv(
		"STACKYARD_AZURE_DATABRICKS_PRIVATE_ENDPOINT_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/privateEndpoints/databricks-private-endpoint",
			subscriptionID,
			resourceGroup,
		),
	)

	client := newDatabricksClient(endpoint, account, subscriptionKey)

	connectorBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Databricks/accessConnectors/%s",
		subscriptionID,
		resourceGroup,
		connectorName,
	)
	workspaceBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Databricks/workspaces/%s",
		subscriptionID,
		resourceGroup,
		workspaceName,
	)
	peeringBase := workspaceBase + "/virtualNetworkPeerings/" + peeringName
	privateEndpointConnectionBase := workspaceBase + "/privateEndpointConnections/" + privateEndpointConnectionName
	privateLinkResourceBase := workspaceBase + "/privateLinkResources/" + privateLinkResourceName

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListProviderOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.Databricks/operations?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateAccessConnector",
			method: http.MethodPut,
			path:   connectorBase + "?api-version=" + databricksAPIVersion,
			payload: map[string]any{
				"location": location,
				"identity": map[string]any{
					"type": "SystemAssigned",
				},
				"tags": map[string]any{
					"environment": "local",
					"service":     "databricks",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetAccessConnector",
			method:   http.MethodGet,
			path:     connectorBase + "?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAccessConnectorsByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Databricks/accessConnectors?api-version=%s", subscriptionID, resourceGroup, databricksAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAccessConnectorsBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.Databricks/accessConnectors?api-version=%s", subscriptionID, databricksAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateAccessConnector",
			method: http.MethodPatch,
			path:   connectorBase + "?api-version=" + databricksAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"environment": "local-updated",
					"service":     "databricks",
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "CreateOrUpdateWorkspace",
			method: http.MethodPut,
			path:   workspaceBase + "?api-version=" + databricksAPIVersion,
			payload: map[string]any{
				"location": location,
				"sku": map[string]any{
					"name": "premium",
				},
				"tags": map[string]any{
					"environment": "local",
					"service":     "databricks",
				},
				"properties": map[string]any{
					"computeMode":            "Hybrid",
					"managedResourceGroupId": managedResourceGroupID,
					"requiredNsgRules":       "AllRules",
					"accessConnector": map[string]any{
						"id":           accessConnectorResourceID,
						"identityType": "SystemAssigned",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetWorkspace",
			method:   http.MethodGet,
			path:     workspaceBase + "?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListWorkspacesByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Databricks/workspaces?api-version=%s", subscriptionID, resourceGroup, databricksAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListWorkspacesBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.Databricks/workspaces?api-version=%s", subscriptionID, databricksAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateWorkspace",
			method: http.MethodPatch,
			path:   workspaceBase + "?api-version=" + databricksAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"environment": "local-updated",
					"service":     "databricks",
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "ListOutboundNetworkDependenciesEndpoints",
			method:   http.MethodGet,
			path:     workspaceBase + "/outboundNetworkDependenciesEndpoints?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateLinkResources",
			method:   http.MethodGet,
			path:     workspaceBase + "/privateLinkResources?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetPrivateLinkResource",
			method:   http.MethodGet,
			path:     privateLinkResourceBase + "?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateVNetPeering",
			method: http.MethodPut,
			path:   peeringBase + "?api-version=" + databricksAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"allowForwardedTraffic":     false,
					"allowGatewayTransit":       false,
					"allowVirtualNetworkAccess": true,
					"remoteVirtualNetwork": map[string]any{
						"id": remoteVirtualNetworkID,
					},
					"useRemoteGateways": false,
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetVNetPeering",
			method:   http.MethodGet,
			path:     peeringBase + "?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "ListVNetPeeringsByWorkspace",
			method:   http.MethodGet,
			path:     workspaceBase + "/virtualNetworkPeerings?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreatePrivateEndpointConnection",
			method: http.MethodPut,
			path:   privateEndpointConnectionBase + "?api-version=" + databricksAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"groupIds": []any{
						privateLinkResourceName,
					},
					"privateEndpoint": map[string]any{
						"id": privateEndpointID,
					},
					"privateLinkServiceConnectionState": map[string]any{
						"status":          "Approved",
						"description":     "Approved in local emulation",
						"actionsRequired": "None",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "GetPrivateEndpointConnection",
			method:   http.MethodGet,
			path:     privateEndpointConnectionBase + "?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateEndpointConnections",
			method:   http.MethodGet,
			path:     workspaceBase + "/privateEndpointConnections?api-version=" + databricksAPIVersion,
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

	cleanup := []struct {
		name     string
		method   string
		path     string
		statuses []int
	}{
		{
			name:     "DeletePrivateEndpointConnection",
			method:   http.MethodDelete,
			path:     privateEndpointConnectionBase + "?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteVNetPeering",
			method:   http.MethodDelete,
			path:     peeringBase + "?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteWorkspace",
			method:   http.MethodDelete,
			path:     workspaceBase + "?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteAccessConnector",
			method:   http.MethodDelete,
			path:     connectorBase + "?api-version=" + databricksAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range cleanup {
		payload, status, err := client.doJSON(ctx, call.method, call.path, nil, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
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
