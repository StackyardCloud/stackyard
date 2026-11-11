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

const firewallAPIVersion = "2025-05-01"
const firewallPrefix = "/azure"

type firewallClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type firewallSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type firewallCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *firewallSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newFirewallClient(endpoint, account, subscriptionKey string) *firewallClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"firewall-2025-05-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&firewallSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &firewallClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *firewallClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return firewallPrefix + path
}

func payloadForMethod(methodPath, location, firewallPolicyID, subnetID, publicIPAddressID string) any {
	switch methodPath {
	case "azure-firewalls/create-or-update":
		return map[string]any{
			"location": location,
			"properties": map[string]any{
				"sku":            map[string]any{"name": "AZFW_VNet", "tier": "Standard"},
				"firewallPolicy": map[string]any{"id": firewallPolicyID},
				"ipConfigurations": []any{
					map[string]any{
						"name": "configuration",
						"properties": map[string]any{
							"subnet":          map[string]any{"id": subnetID},
							"publicIPAddress": map[string]any{"id": publicIPAddressID},
						},
					},
				},
				"threatIntelMode": "Alert",
			},
			"tags": map[string]any{"env": "local"},
		}
	case "azure-firewalls/packet-capture", "azure-firewalls/packet-capture-operation":
		return map[string]any{
			"operation":                "Start",
			"sasUrl":                   "https://examplestorage.blob.core.windows.net/captures/firewallcapture.cap?sig=stub",
			"durationInSeconds":        60,
			"numberOfPacketsToCapture": 1000,
			"protocol":                 "Any",
			"fileName":                 "capture.cap",
		}
	case "azure-firewalls/update-tags", "firewall-policies/update-tags":
		return map[string]any{"tags": map[string]any{"env": "local", "owner": "network"}}
	case "firewall-policies/create-or-update":
		return map[string]any{
			"location": location,
			"properties": map[string]any{
				"sku":             map[string]any{"tier": "Standard"},
				"threatIntelMode": "Alert",
			},
			"tags": map[string]any{"env": "local"},
		}
	case "firewall-policy-drafts/create-or-update":
		return map[string]any{
			"location":   location,
			"properties": map[string]any{"threatIntelMode": "Alert"},
		}
	case "firewall-policy-idps-signatures/list":
		return map[string]any{
			"resultsPerPage": 25,
			"skip":           0,
			"search":         "sql",
			"orderBy":        map[string]any{"field": "Severity", "order": "Ascending"},
		}
	case "firewall-policy-idps-signatures-filter-values/list":
		return map[string]any{"filterName": "severity"}
	case "firewall-policy-idps-signatures-overrides/patch", "firewall-policy-idps-signatures-overrides/put":
		return map[string]any{
			"properties": map[string]any{
				"signatureOverrides": []any{
					map[string]any{"id": "200001", "mode": "Alert"},
				},
			},
		}
	case "firewall-policy-rule-collection-group-drafts/create-or-update", "firewall-policy-rule-collection-groups/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"priority": 100,
				"ruleCollections": []any{
					map[string]any{
						"name":               "rcg-default",
						"priority":           100,
						"ruleCollectionType": "FirewallPolicyFilterRuleCollection",
						"action":             map[string]any{"type": "Allow"},
						"rules":              []any{},
					},
				},
			},
		}
	}
	return map[string]any{}
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_FIREWALL_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_FIREWALL_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-firewall")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	azureFirewallName := getenv("STACKYARD_AZURE_FIREWALL_NAME", "azfw01")
	firewallPolicyName := getenv("STACKYARD_AZURE_FIREWALL_POLICY_NAME", "fwpolicy01")
	ruleCollectionGroupName := getenv("STACKYARD_AZURE_FIREWALL_RULE_COLLECTION_GROUP_NAME", "rcg01")
	webCategoryName := getenv("STACKYARD_AZURE_FIREWALL_WEB_CATEGORY_NAME", "General")

	firewallPolicyID := getenv(
		"STACKYARD_AZURE_FIREWALL_POLICY_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/firewallPolicies/%s", subscriptionID, resourceGroupName, firewallPolicyName),
	)
	subnetID := getenv(
		"STACKYARD_AZURE_FIREWALL_SUBNET_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/vnet-firewall/subnets/AzureFirewallSubnet", subscriptionID, resourceGroupName),
	)
	publicIPAddressID := getenv(
		"STACKYARD_AZURE_FIREWALL_PUBLIC_IP_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/pip-firewall", subscriptionID, resourceGroupName),
	)

	client := newFirewallClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":          subscriptionID,
		"resourceGroupName":       resourceGroupName,
		"azureFirewallName":       azureFirewallName,
		"firewallPolicyName":      firewallPolicyName,
		"ruleCollectionGroupName": ruleCollectionGroupName,
		"name":                    webCategoryName,
	}

	calls := []firewallCall{
		{
			name:     "azure_firewall_fqdn_tags_list_all",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/azureFirewallFqdnTags?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "azure_firewalls_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/azureFirewalls/{azureFirewallName}?api-version=2025-05-01",
			payload:  payloadForMethod("azure-firewalls/create-or-update", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200, 201},
		},
		{
			name:     "azure_firewalls_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/azureFirewalls/{azureFirewallName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "azure_firewalls_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/azureFirewalls/{azureFirewallName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "azure_firewalls_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/azureFirewalls?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "azure_firewalls_list_all",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/azureFirewalls?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "azure_firewalls_list_learned_prefixes",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/azureFirewalls/{azureFirewallName}/learnedIPPrefixes?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "azure_firewalls_packet_capture",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/azureFirewalls/{azureFirewallName}/packetCapture?api-version=2025-05-01",
			payload:  payloadForMethod("azure-firewalls/packet-capture", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{202},
		},
		{
			name:     "azure_firewalls_packet_capture_operation",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/azureFirewalls/{azureFirewallName}/packetCaptureOperation?api-version=2025-05-01",
			payload:  payloadForMethod("azure-firewalls/packet-capture-operation", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200, 202},
		},
		{
			name:     "azure_firewalls_update_tags",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/azureFirewalls/{azureFirewallName}?api-version=2025-05-01",
			payload:  payloadForMethod("azure-firewalls/update-tags", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200, 202},
		},
		{
			name:     "firewall_policies_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}?api-version=2025-05-01",
			payload:  payloadForMethod("firewall-policies/create-or-update", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200, 201},
		},
		{
			name:     "firewall_policies_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "firewall_policies_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "firewall_policies_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "firewall_policies_list_all",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/firewallPolicies?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "firewall_policies_update_tags",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}?api-version=2025-05-01",
			payload:  payloadForMethod("firewall-policies/update-tags", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200},
		},
		{
			name:     "firewall_policy_deployments_deploy",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/deploy?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{202},
		},
		{
			name:     "firewall_policy_drafts_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/firewallPolicyDrafts/default?api-version=2025-05-01",
			payload:  payloadForMethod("firewall-policy-drafts/create-or-update", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200, 201},
		},
		{
			name:     "firewall_policy_drafts_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/firewallPolicyDrafts/default?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 204},
		},
		{
			name:     "firewall_policy_drafts_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/firewallPolicyDrafts/default?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "firewall_policy_idps_signatures_list",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/listIdpsSignatures?api-version=2025-05-01",
			payload:  payloadForMethod("firewall-policy-idps-signatures/list", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200},
		},
		{
			name:     "firewall_policy_idps_signatures_filter_values_list",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/listIdpsFilterOptions?api-version=2025-05-01",
			payload:  payloadForMethod("firewall-policy-idps-signatures-filter-values/list", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200},
		},
		{
			name:     "firewall_policy_idps_signatures_overrides_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/signatureOverrides/default?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "firewall_policy_idps_signatures_overrides_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/signatureOverrides?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "firewall_policy_idps_signatures_overrides_patch",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/signatureOverrides/default?api-version=2025-05-01",
			payload:  payloadForMethod("firewall-policy-idps-signatures-overrides/patch", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200},
		},
		{
			name:     "firewall_policy_idps_signatures_overrides_put",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/signatureOverrides/default?api-version=2025-05-01",
			payload:  payloadForMethod("firewall-policy-idps-signatures-overrides/put", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200},
		},
		{
			name:     "firewall_policy_rule_collection_group_drafts_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/ruleCollectionGroups/{ruleCollectionGroupName}/ruleCollectionGroupDrafts/default?api-version=2025-05-01",
			payload:  payloadForMethod("firewall-policy-rule-collection-group-drafts/create-or-update", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200, 201},
		},
		{
			name:     "firewall_policy_rule_collection_group_drafts_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/ruleCollectionGroups/{ruleCollectionGroupName}/ruleCollectionGroupDrafts/default?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 204},
		},
		{
			name:     "firewall_policy_rule_collection_group_drafts_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/ruleCollectionGroups/{ruleCollectionGroupName}/ruleCollectionGroupDrafts/default?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "firewall_policy_rule_collection_groups_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/ruleCollectionGroups/{ruleCollectionGroupName}?api-version=2025-05-01",
			payload:  payloadForMethod("firewall-policy-rule-collection-groups/create-or-update", location, firewallPolicyID, subnetID, publicIPAddressID),
			statuses: []int{200, 201},
		},
		{
			name:     "firewall_policy_rule_collection_groups_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/ruleCollectionGroups/{ruleCollectionGroupName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "firewall_policy_rule_collection_groups_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/ruleCollectionGroups/{ruleCollectionGroupName}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "firewall_policy_rule_collection_groups_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/ruleCollectionGroups?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "web_categories_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/azureWebCategories/{name}?api-version=2025-05-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "web_categories_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/azureWebCategories?api-version=2025-05-01",
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
