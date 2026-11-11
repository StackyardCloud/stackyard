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

const expressRouteAPIVersion = "2025-05-01"
const expressRoutePrefix = "/azure"

type expressRouteClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type expressRouteSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type expressRouteCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *expressRouteSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newExpressRouteClient(endpoint, account, subscriptionKey string) *expressRouteClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"expressroute-2025-05-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&expressRouteSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &expressRouteClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *expressRouteClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return expressRoutePrefix + path
}

func payloadForMethod(methodPath, location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID string) any {
	switch methodPath {
	case "express-route-circuit-authorizations/create-or-update":
		return map[string]any{"properties": map[string]any{}}
	case "express-route-circuit-connections/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"addressPrefix":                  "10.10.0.0/29",
				"peerExpressRouteCircuitPeering": map[string]any{"id": peerCircuitPeeringID},
			},
		}
	case "express-route-circuit-peerings/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"peeringType":                "AzurePrivatePeering",
				"peerASN":                    100,
				"primaryPeerAddressPrefix":   "30.0.0.0/30",
				"secondaryPeerAddressPrefix": "30.0.0.4/30",
				"vlanId":                     200,
			},
		}
	case "express-route-circuits/create-or-update":
		return map[string]any{
			"location": location,
			"sku":      map[string]any{"name": "Standard_MeteredData", "tier": "Standard", "family": "MeteredData"},
			"properties": map[string]any{
				"allowClassicOperations": false,
				"serviceProviderProperties": map[string]any{
					"serviceProviderName": "providerName",
					"peeringLocation":     "SiliconValley",
					"bandwidthInMbps":     200,
				},
			},
		}
	case "express-route-circuits/update-tags":
		return map[string]any{"tags": map[string]any{"env": "local", "updated": "true"}}
	case "express-route-cross-connection-peerings/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"peeringType":                "AzurePrivatePeering",
				"peerASN":                    200,
				"primaryPeerAddressPrefix":   "40.0.0.0/30",
				"secondaryPeerAddressPrefix": "40.0.0.4/30",
				"vlanId":                     300,
			},
		}
	case "express-route-cross-connections/create-or-update":
		return map[string]any{
			"location":   location,
			"properties": map[string]any{"serviceProviderProvisioningState": "NotProvisioned"},
		}
	case "express-route-cross-connections/update-tags":
		return map[string]any{"tags": map[string]any{"env": "local", "team": "network"}}
	case "express-route-port-authorizations/create-or-update":
		return map[string]any{"properties": map[string]any{}}
	case "express-route-ports/create-or-update":
		return map[string]any{
			"location": location,
			"properties": map[string]any{
				"encapsulation":              "Dot1Q",
				"peeringLocation":            "SiliconValley",
				"bandwidthInGbps":            10,
				"billingType":                "MeteredData",
				"mtu":                        "1500",
				"ethertype":                  "0x0800",
				"identity":                   map[string]any{"type": "SystemAssigned"},
				"resourceGuid":               "00000000-0000-0000-0000-000000000001",
				"expressRouteLinkAdminState": "Enabled",
			},
		}
	case "express-route-ports/generate-loa":
		return map[string]any{"customerName": "Contoso"}
	case "express-route-ports/update-tags":
		return map[string]any{"tags": map[string]any{"env": "local", "purpose": "test"}}
	case "route-filter-rules/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"access":              "Allow",
				"routeFilterRuleType": "Community",
				"communities":         []any{"12076:5030", "12076:5040"},
			},
		}
	case "route-filters/create-or-update":
		return map[string]any{"location": location, "tags": map[string]any{"env": "local"}}
	case "route-filters/update-tags":
		return map[string]any{"tags": map[string]any{"env": "local", "owner": "platform"}}
	case "service-endpoint-policies/create-or-update":
		return map[string]any{
			"location": location,
			"properties": map[string]any{
				"serviceAlias": "Microsoft.Storage",
				"serviceEndpointPolicyDefinitions": []any{
					map[string]any{"id": serviceEndpointPolicyDefinitionID},
				},
			},
		}
	case "service-endpoint-policies/update-tags":
		return map[string]any{"tags": map[string]any{"env": "local", "managed": "true"}}
	case "service-endpoint-policy-definitions/create-or-update":
		return map[string]any{
			"name": "policyDef01",
			"properties": map[string]any{
				"description":      "local definition",
				"service":          "Microsoft.Storage",
				"serviceResources": []any{serviceResourceID},
			},
		}
	}
	return map[string]any{}
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_EXPRESSROUTE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_EXPRESSROUTE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-expressroute")
	location := getenv("STACKYARD_AZURE_LOCATION", "westus")
	circuitName := getenv("STACKYARD_AZURE_EXPRESSROUTE_CIRCUIT_NAME", "circuit01")
	authorizationName := getenv("STACKYARD_AZURE_EXPRESSROUTE_AUTHORIZATION_NAME", "authorization01")
	peeringName := getenv("STACKYARD_AZURE_EXPRESSROUTE_PEERING_NAME", "AzurePrivatePeering")
	connectionName := getenv("STACKYARD_AZURE_EXPRESSROUTE_CONNECTION_NAME", "connection01")
	devicePath := getenv("STACKYARD_AZURE_EXPRESSROUTE_DEVICE_PATH", "primary")
	crossConnectionName := getenv("STACKYARD_AZURE_EXPRESSROUTE_CROSS_CONNECTION_NAME", "crossconnection01")
	expressRoutePortName := getenv("STACKYARD_AZURE_EXPRESSROUTE_PORT_NAME", "port01")
	linkName := getenv("STACKYARD_AZURE_EXPRESSROUTE_LINK_NAME", "link1")
	locationName := getenv("STACKYARD_AZURE_EXPRESSROUTE_LOCATION_NAME", "SiliconValley")
	providerPortName := getenv("STACKYARD_AZURE_EXPRESSROUTE_PROVIDER_PORT_NAME", "providerport01")
	routeFilterName := getenv("STACKYARD_AZURE_EXPRESSROUTE_ROUTE_FILTER_NAME", "routefilter01")
	ruleName := getenv("STACKYARD_AZURE_EXPRESSROUTE_RULE_NAME", "rule01")
	serviceEndpointPolicyName := getenv("STACKYARD_AZURE_EXPRESSROUTE_SERVICE_ENDPOINT_POLICY_NAME", "policy01")
	serviceEndpointPolicyDefinitionName := getenv("STACKYARD_AZURE_EXPRESSROUTE_SERVICE_ENDPOINT_POLICY_DEFINITION_NAME", "definition01")

	peerCircuitPeeringID := getenv(
		"STACKYARD_AZURE_EXPRESSROUTE_PEER_CIRCUIT_PEERING_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/expressRouteCircuits/%s/peerings/%s", subscriptionID, resourceGroupName, circuitName, peeringName),
	)
	serviceEndpointPolicyDefinitionID := getenv(
		"STACKYARD_AZURE_EXPRESSROUTE_SERVICE_ENDPOINT_POLICY_DEFINITION_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/serviceEndpointPolicies/%s/serviceEndpointPolicyDefinitions/%s", subscriptionID, resourceGroupName, serviceEndpointPolicyName, serviceEndpointPolicyDefinitionName),
	)
	serviceResourceID := getenv(
		"STACKYARD_AZURE_EXPRESSROUTE_SERVICE_RESOURCE_ID",
		"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-expressroute/providers/Microsoft.Storage/storageAccounts/storage01",
	)

	client := newExpressRouteClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":                      subscriptionID,
		"resourceGroupName":                   resourceGroupName,
		"circuitName":                         circuitName,
		"authorizationName":                   authorizationName,
		"peeringName":                         peeringName,
		"connectionName":                      connectionName,
		"devicePath":                          devicePath,
		"crossConnectionName":                 crossConnectionName,
		"expressRoutePortName":                expressRoutePortName,
		"linkName":                            linkName,
		"locationName":                        locationName,
		"providerport":                        providerPortName,
		"routeFilterName":                     routeFilterName,
		"ruleName":                            ruleName,
		"serviceEndpointPolicyName":           serviceEndpointPolicyName,
		"serviceEndpointPolicyDefinitionName": serviceEndpointPolicyDefinitionName,
	}

	calls := []expressRouteCall{
		{
			name:     "bgp_service_communities_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/bgpServiceCommunities?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuit_authorizations_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/authorizations/{authorizationName}?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-circuit-authorizations/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200, 201},
		},
		{
			name:     "express_route_circuit_authorizations_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/authorizations/{authorizationName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "express_route_circuit_authorizations_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/authorizations/{authorizationName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuit_authorizations_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/authorizations?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuit_connections_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}/connections/{connectionName}?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-circuit-connections/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200, 201},
		},
		{
			name:     "express_route_circuit_connections_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}/connections/{connectionName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "express_route_circuit_connections_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}/connections/{connectionName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuit_connections_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}/connections?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuit_peerings_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-circuit-peerings/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200, 201},
		},
		{
			name:     "express_route_circuit_peerings_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "express_route_circuit_peerings_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuit_peerings_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuits_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-circuits/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200, 201},
		},
		{
			name:     "express_route_circuits_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "express_route_circuits_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuits_get_peering_stats",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}/stats?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuits_get_stats",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/stats?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuits_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuits_list_all",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/expressRouteCircuits?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_circuits_list_arp_table",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}/arpTables/{devicePath}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "express_route_circuits_list_routes_table",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}/routeTables/{devicePath}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "express_route_circuits_list_routes_table_summary",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}/routeTablesSummary/{devicePath}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "express_route_circuits_update_tags",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-circuits/update-tags", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200},
		},
		{
			name:     "express_route_cross_connection_peerings_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections/{crossConnectionName}/peerings/{peeringName}?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-cross-connection-peerings/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200, 201},
		},
		{
			name:     "express_route_cross_connection_peerings_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections/{crossConnectionName}/peerings/{peeringName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "express_route_cross_connection_peerings_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections/{crossConnectionName}/peerings/{peeringName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_cross_connection_peerings_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections/{crossConnectionName}/peerings?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_cross_connections_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections/{crossConnectionName}?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-cross-connections/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200},
		},
		{
			name:     "express_route_cross_connections_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections/{crossConnectionName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_cross_connections_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/expressRouteCrossConnections?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_cross_connections_list_arp_table",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections/{crossConnectionName}/peerings/{peeringName}/arpTables/{devicePath}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "express_route_cross_connections_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_cross_connections_list_routes_table",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections/{crossConnectionName}/peerings/{peeringName}/routeTables/{devicePath}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "express_route_cross_connections_list_routes_table_summary",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections/{crossConnectionName}/peerings/{peeringName}/routeTablesSummary/{devicePath}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "express_route_cross_connections_update_tags",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections/{crossConnectionName}?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-cross-connections/update-tags", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200},
		},
		{
			name:     "express_route_links_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ExpressRoutePorts/{expressRoutePortName}/links/{linkName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_links_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ExpressRoutePorts/{expressRoutePortName}/links?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_port_authorizations_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRoutePorts/{expressRoutePortName}/authorizations/{authorizationName}?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-port-authorizations/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200, 201},
		},
		{
			name:     "express_route_port_authorizations_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRoutePorts/{expressRoutePortName}/authorizations/{authorizationName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "express_route_port_authorizations_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRoutePorts/{expressRoutePortName}/authorizations/{authorizationName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_port_authorizations_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRoutePorts/{expressRoutePortName}/authorizations?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_ports_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ExpressRoutePorts/{expressRoutePortName}?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-ports/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200, 201},
		},
		{
			name:     "express_route_ports_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ExpressRoutePorts/{expressRoutePortName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "express_route_ports_generate_loa",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ExpressRoutePorts/{expressRoutePortName}/generateLoa?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-ports/generate-loa", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200},
		},
		{
			name:     "express_route_ports_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ExpressRoutePorts/{expressRoutePortName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_ports_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/ExpressRoutePorts?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_ports_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ExpressRoutePorts?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_ports_update_tags",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ExpressRoutePorts/{expressRoutePortName}?api-version=2025-05-01",
			payload:  payloadForMethod("express-route-ports/update-tags", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200},
		},
		{
			name:     "express_route_ports_locations_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/ExpressRoutePortsLocations/{locationName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_ports_locations_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/ExpressRoutePortsLocations?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_provider_port_express_route_provider_port",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/expressRouteProviderPorts/{providerport}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_provider_ports_location_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/expressRouteProviderPorts?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "express_route_service_providers_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/expressRouteServiceProviders?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "peer_express_route_circuit_connections_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}/peerConnections/{connectionName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "peer_express_route_circuit_connections_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}/peerings/{peeringName}/peerConnections?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "route_filter_rules_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules/{ruleName}?api-version=2025-05-01",
			payload:  payloadForMethod("route-filter-rules/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200, 201},
		},
		{
			name:     "route_filter_rules_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules/{ruleName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "route_filter_rules_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules/{ruleName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "route_filter_rules_list_by_route_filter",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "route_filters_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}?api-version=2025-05-01",
			payload:  payloadForMethod("route-filters/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200, 201},
		},
		{
			name:     "route_filters_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "route_filters_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "route_filters_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/routeFilters?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "route_filters_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "route_filters_update_tags",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}?api-version=2025-05-01",
			payload:  payloadForMethod("route-filters/update-tags", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200},
		},
		{
			name:     "service_endpoint_policies_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/serviceEndpointPolicies/{serviceEndpointPolicyName}?api-version=2025-05-01",
			payload:  payloadForMethod("service-endpoint-policies/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200, 201},
		},
		{
			name:     "service_endpoint_policies_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/serviceEndpointPolicies/{serviceEndpointPolicyName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "service_endpoint_policies_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/serviceEndpointPolicies/{serviceEndpointPolicyName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "service_endpoint_policies_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/serviceEndpointPolicies?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "service_endpoint_policies_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/serviceEndpointPolicies?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "service_endpoint_policies_update_tags",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/serviceEndpointPolicies/{serviceEndpointPolicyName}?api-version=2025-05-01",
			payload:  payloadForMethod("service-endpoint-policies/update-tags", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200},
		},
		{
			name:     "service_endpoint_policy_definitions_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/serviceEndpointPolicies/{serviceEndpointPolicyName}/serviceEndpointPolicyDefinitions/{serviceEndpointPolicyDefinitionName}?api-version=2025-05-01",
			payload:  payloadForMethod("service-endpoint-policy-definitions/create-or-update", location, peerCircuitPeeringID, serviceEndpointPolicyDefinitionID, serviceResourceID),
			statuses: []int{200, 201},
		},
		{
			name:     "service_endpoint_policy_definitions_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/serviceEndpointPolicies/{serviceEndpointPolicyName}/serviceEndpointPolicyDefinitions/{serviceEndpointPolicyDefinitionName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "service_endpoint_policy_definitions_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/serviceEndpointPolicies/{serviceEndpointPolicyName}/serviceEndpointPolicyDefinitions/{serviceEndpointPolicyDefinitionName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "service_endpoint_policy_definitions_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/serviceEndpointPolicies/{serviceEndpointPolicyName}/serviceEndpointPolicyDefinitions?api-version=2025-05-01",
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
