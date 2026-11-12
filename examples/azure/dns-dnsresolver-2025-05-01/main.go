package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const dnsResolverAPIVersion = "2025-05-01"
const dnsResolverPrefix = "/azure"

type dnsResolverClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dnsResolverSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type dnsResolverCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *dnsResolverSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDNSResolverClient(endpoint, account, subscriptionKey string) *dnsResolverClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"dns-dnsresolver-2025-05-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&dnsResolverSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &dnsResolverClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dnsResolverClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return dnsResolverPrefix + path
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-dnsresolver")
	location := getenv("STACKYARD_AZURE_LOCATION", "westus2")

	dnsResolverName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_NAME", "dnsresolver01")
	inboundEndpointName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_INBOUND_ENDPOINT_NAME", "inbound01")
	outboundEndpointName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_OUTBOUND_ENDPOINT_NAME", "outbound01")
	dnsForwardingRulesetName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_FORWARDING_RULESET_NAME", "ruleset01")
	forwardingRuleName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_FORWARDING_RULE_NAME", "rule01")
	virtualNetworkLinkName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_RULESET_VNET_LINK_NAME", "rulesetvnetlink01")
	dnsResolverDomainListName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_DOMAIN_LIST_NAME", "domainlist01")
	dnsResolverPolicyName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_POLICY_NAME", "policy01")
	dnsResolverPolicyVirtualNetworkLinkName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_POLICY_VNET_LINK_NAME", "policyvnetlink01")
	dnsSecurityRuleName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_SECURITY_RULE_NAME", "securityrule01")
	virtualNetworkName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_VIRTUAL_NETWORK_NAME", "vnet-dnsresolver")

	resolverVirtualNetworkID := getenv(
		"STACKYARD_AZURE_DNS_DNSRESOLVER_VIRTUAL_NETWORK_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s", subscriptionID, resourceGroupName, virtualNetworkName),
	)
	inboundEndpointSubnetID := getenv(
		"STACKYARD_AZURE_DNS_DNSRESOLVER_INBOUND_SUBNET_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/inbound", subscriptionID, resourceGroupName, virtualNetworkName),
	)
	outboundEndpointSubnetID := getenv(
		"STACKYARD_AZURE_DNS_DNSRESOLVER_OUTBOUND_SUBNET_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/outbound", subscriptionID, resourceGroupName, virtualNetworkName),
	)
	rulesetVirtualNetworkID := getenv(
		"STACKYARD_AZURE_DNS_DNSRESOLVER_RULESET_VIRTUAL_NETWORK_ID",
		resolverVirtualNetworkID,
	)
	policyVirtualNetworkID := getenv(
		"STACKYARD_AZURE_DNS_DNSRESOLVER_POLICY_VIRTUAL_NETWORK_ID",
		resolverVirtualNetworkID,
	)
	bulkStorageURL := getenv(
		"STACKYARD_AZURE_DNS_DNSRESOLVER_BULK_STORAGE_URL",
		"https://examplestorage.blob.core.windows.net/dnsresolver/domain-list.txt",
	)
	targetDNSServerIP := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_TARGET_DNS_SERVER_IP", "10.0.0.10")
	targetDNSServerPort := getenvInt("STACKYARD_AZURE_DNS_DNSRESOLVER_TARGET_DNS_SERVER_PORT", 53)
	forwardingDomainName := getenv("STACKYARD_AZURE_DNS_DNSRESOLVER_FORWARDING_DOMAIN_NAME", "contoso.com.")

	outboundEndpointResourceID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/dnsResolvers/%s/outboundEndpoints/%s",
		subscriptionID,
		resourceGroupName,
		dnsResolverName,
		outboundEndpointName,
	)
	domainListResourceID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/dnsResolverDomainLists/%s",
		subscriptionID,
		resourceGroupName,
		dnsResolverDomainListName,
	)

	client := newDNSResolverClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":                          subscriptionID,
		"resourceGroupName":                       resourceGroupName,
		"dnsForwardingRulesetName":                dnsForwardingRulesetName,
		"dnsResolverDomainListName":               dnsResolverDomainListName,
		"dnsResolverPolicyName":                   dnsResolverPolicyName,
		"dnsResolverPolicyVirtualNetworkLinkName": dnsResolverPolicyVirtualNetworkLinkName,
		"dnsResolverName":                         dnsResolverName,
		"dnsSecurityRuleName":                     dnsSecurityRuleName,
		"forwardingRuleName":                      forwardingRuleName,
		"inboundEndpointName":                     inboundEndpointName,
		"outboundEndpointName":                    outboundEndpointName,
		"virtualNetworkLinkName":                  virtualNetworkLinkName,
		"virtualNetworkName":                      virtualNetworkName,
	}

	payloadDNSResolverCreate := map[string]any{
		"location": location,
		"properties": map[string]any{
			"virtualNetwork": map[string]any{
				"id": resolverVirtualNetworkID,
			},
		},
		"tags": map[string]any{"env": "local", "service": "dnsresolver"},
	}

	payloadDNSResolverUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
	}

	payloadInboundEndpointCreate := map[string]any{
		"location": location,
		"properties": map[string]any{
			"ipConfigurations": []any{
				map[string]any{
					"privateIpAllocationMethod": "Dynamic",
					"subnet":                    map[string]any{"id": inboundEndpointSubnetID},
				},
			},
		},
		"tags": map[string]any{"env": "local"},
	}

	payloadInboundEndpointUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
	}

	payloadOutboundEndpointCreate := map[string]any{
		"location": location,
		"properties": map[string]any{
			"subnet": map[string]any{"id": outboundEndpointSubnetID},
		},
		"tags": map[string]any{"env": "local"},
	}

	payloadOutboundEndpointUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
	}

	payloadDNSForwardingRulesetCreate := map[string]any{
		"location": location,
		"properties": map[string]any{
			"dnsResolverOutboundEndpoints": []any{
				map[string]any{"id": outboundEndpointResourceID},
			},
		},
		"tags": map[string]any{"env": "local"},
	}

	payloadDNSForwardingRulesetUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
	}

	payloadForwardingRuleCreate := map[string]any{
		"properties": map[string]any{
			"domainName":          forwardingDomainName,
			"forwardingRuleState": "Enabled",
			"metadata":            map[string]any{"env": "local"},
			"targetDnsServers": []any{
				map[string]any{"ipAddress": targetDNSServerIP, "port": targetDNSServerPort},
			},
		},
	}

	payloadForwardingRuleUpdate := map[string]any{
		"properties": map[string]any{
			"metadata": map[string]any{"env": "local-updated"},
			"targetDnsServers": []any{
				map[string]any{"ipAddress": targetDNSServerIP, "port": targetDNSServerPort},
			},
		},
	}

	payloadRulesetVirtualNetworkLinkCreate := map[string]any{
		"properties": map[string]any{
			"metadata":       map[string]any{"env": "local"},
			"virtualNetwork": map[string]any{"id": rulesetVirtualNetworkID},
		},
	}

	payloadRulesetVirtualNetworkLinkUpdate := map[string]any{
		"properties": map[string]any{
			"metadata": map[string]any{"env": "local-updated"},
		},
	}

	payloadDomainListCreate := map[string]any{
		"location": location,
		"properties": map[string]any{
			"domains": []any{"contoso.com", "internal.contoso.com"},
		},
		"tags": map[string]any{"env": "local"},
	}

	payloadDomainListUpdate := map[string]any{
		"properties": map[string]any{
			"domains": []any{"contoso.com", "corp.contoso.com"},
		},
		"tags": map[string]any{"env": "local-updated"},
	}

	payloadDomainListBulk := map[string]any{
		"properties": map[string]any{
			"action":     "Upload",
			"storageUrl": bulkStorageURL,
		},
	}

	payloadPolicyCreate := map[string]any{
		"location": location,
		"tags":     map[string]any{"env": "local"},
	}

	payloadPolicyUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
	}

	payloadPolicyVirtualNetworkLinkCreate := map[string]any{
		"location": location,
		"properties": map[string]any{
			"virtualNetwork": map[string]any{"id": policyVirtualNetworkID},
		},
		"tags": map[string]any{"env": "local"},
	}

	payloadPolicyVirtualNetworkLinkUpdate := map[string]any{
		"tags": map[string]any{"env": "local-updated"},
	}

	payloadSecurityRuleCreate := map[string]any{
		"location": location,
		"properties": map[string]any{
			"action": map[string]any{"actionType": "Block"},
			"dnsResolverDomainLists": []any{
				map[string]any{"id": domainListResourceID},
			},
			"dnsSecurityRuleState": "Enabled",
			"priority":             100,
		},
		"tags": map[string]any{"env": "local"},
	}

	payloadSecurityRuleUpdate := map[string]any{
		"properties": map[string]any{
			"action": map[string]any{"actionType": "Allow"},
			"dnsResolverDomainLists": []any{
				map[string]any{"id": domainListResourceID},
			},
			"dnsSecurityRuleState": "Enabled",
			"priority":             110,
		},
		"tags": map[string]any{"env": "local-updated"},
	}

	calls := []dnsResolverCall{
		{
			name:     "dns_resolvers_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}?api-version=2025-05-01",
			payload:  payloadDNSResolverCreate,
			statuses: []int{200, 201, 202},
		},
		{
			name:     "dns_resolvers_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolvers_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/dnsResolvers?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolvers_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolvers_list_by_virtual_network",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/listDnsResolvers?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolvers_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}?api-version=2025-05-01",
			payload:  payloadDNSResolverUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "inbound_endpoints_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/inboundEndpoints/{inboundEndpointName}?api-version=2025-05-01",
			payload:  payloadInboundEndpointCreate,
			statuses: []int{200, 201, 202},
		},
		{
			name:     "inbound_endpoints_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/inboundEndpoints/{inboundEndpointName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "inbound_endpoints_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/inboundEndpoints?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "inbound_endpoints_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/inboundEndpoints/{inboundEndpointName}?api-version=2025-05-01",
			payload:  payloadInboundEndpointUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "outbound_endpoints_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/outboundEndpoints/{outboundEndpointName}?api-version=2025-05-01",
			payload:  payloadOutboundEndpointCreate,
			statuses: []int{200, 201, 202},
		},
		{
			name:     "outbound_endpoints_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/outboundEndpoints/{outboundEndpointName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "outbound_endpoints_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/outboundEndpoints?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "outbound_endpoints_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/outboundEndpoints/{outboundEndpointName}?api-version=2025-05-01",
			payload:  payloadOutboundEndpointUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "dns_forwarding_rulesets_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}?api-version=2025-05-01",
			payload:  payloadDNSForwardingRulesetCreate,
			statuses: []int{200, 201, 202},
		},
		{
			name:     "dns_forwarding_rulesets_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_forwarding_rulesets_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/dnsForwardingRulesets?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_forwarding_rulesets_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_forwarding_rulesets_list_by_virtual_network",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/listDnsForwardingRulesets?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_forwarding_rulesets_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}?api-version=2025-05-01",
			payload:  payloadDNSForwardingRulesetUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "forwarding_rules_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/forwardingRules/{forwardingRuleName}?api-version=2025-05-01",
			payload:  payloadForwardingRuleCreate,
			statuses: []int{200, 201},
		},
		{
			name:     "forwarding_rules_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/forwardingRules/{forwardingRuleName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "forwarding_rules_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/forwardingRules?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "forwarding_rules_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/forwardingRules/{forwardingRuleName}?api-version=2025-05-01",
			payload:  payloadForwardingRuleUpdate,
			statuses: []int{200},
		},
		{
			name:     "virtual_network_links_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2025-05-01",
			payload:  payloadRulesetVirtualNetworkLinkCreate,
			statuses: []int{200, 201, 202},
		},
		{
			name:     "virtual_network_links_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "virtual_network_links_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/virtualNetworkLinks?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "virtual_network_links_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2025-05-01",
			payload:  payloadRulesetVirtualNetworkLinkUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "dns_resolver_domain_lists_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists/{dnsResolverDomainListName}?api-version=2025-05-01",
			payload:  payloadDomainListCreate,
			statuses: []int{200, 201},
		},
		{
			name:     "dns_resolver_domain_lists_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists/{dnsResolverDomainListName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolver_domain_lists_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/dnsResolverDomainLists?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolver_domain_lists_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolver_domain_lists_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists/{dnsResolverDomainListName}?api-version=2025-05-01",
			payload:  payloadDomainListUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "dns_resolver_domain_lists_bulk",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists/{dnsResolverDomainListName}/bulk?api-version=2025-05-01",
			payload:  payloadDomainListBulk,
			statuses: []int{200, 202},
		},
		{
			name:     "dns_resolver_policies_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}?api-version=2025-05-01",
			payload:  payloadPolicyCreate,
			statuses: []int{200, 201},
		},
		{
			name:     "dns_resolver_policies_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolver_policies_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/dnsResolverPolicies?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolver_policies_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolver_policies_list_by_virtual_network",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/listDnsResolverPolicies?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolver_policies_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}?api-version=2025-05-01",
			payload:  payloadPolicyUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "dns_resolver_policy_virtual_network_links_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/virtualNetworkLinks/{dnsResolverPolicyVirtualNetworkLinkName}?api-version=2025-05-01",
			payload:  payloadPolicyVirtualNetworkLinkCreate,
			statuses: []int{200, 201},
		},
		{
			name:     "dns_resolver_policy_virtual_network_links_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/virtualNetworkLinks/{dnsResolverPolicyVirtualNetworkLinkName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolver_policy_virtual_network_links_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/virtualNetworkLinks?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_resolver_policy_virtual_network_links_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/virtualNetworkLinks/{dnsResolverPolicyVirtualNetworkLinkName}?api-version=2025-05-01",
			payload:  payloadPolicyVirtualNetworkLinkUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "dns_security_rules_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/dnsSecurityRules/{dnsSecurityRuleName}?api-version=2025-05-01",
			payload:  payloadSecurityRuleCreate,
			statuses: []int{200, 201},
		},
		{
			name:     "dns_security_rules_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/dnsSecurityRules/{dnsSecurityRuleName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_security_rules_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/dnsSecurityRules?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "dns_security_rules_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/dnsSecurityRules/{dnsSecurityRuleName}?api-version=2025-05-01",
			payload:  payloadSecurityRuleUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "dns_security_rules_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/dnsSecurityRules/{dnsSecurityRuleName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{202, 204},
		},
		{
			name:     "dns_resolver_policy_virtual_network_links_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/virtualNetworkLinks/{dnsResolverPolicyVirtualNetworkLinkName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{202, 204},
		},
		{
			name:     "dns_resolver_policies_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{202, 204},
		},
		{
			name:     "dns_resolver_domain_lists_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists/{dnsResolverDomainListName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{202, 204},
		},
		{
			name:     "virtual_network_links_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "forwarding_rules_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/forwardingRules/{forwardingRuleName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 204},
		},
		{
			name:     "dns_forwarding_rulesets_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "inbound_endpoints_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/inboundEndpoints/{inboundEndpointName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "outbound_endpoints_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/outboundEndpoints/{outboundEndpointName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "dns_resolvers_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
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

func getenvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
