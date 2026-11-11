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

const privateDNSAPIVersion = "2018-09-01"
const privateDNSPrefix = "/azure"

type privateDNSClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type privateDNSSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type privateDNSCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *privateDNSSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newPrivateDNSClient(endpoint, account, subscriptionKey string) *privateDNSClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"dns-privatedns-2018-09-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&privateDNSSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &privateDNSClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *privateDNSClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return privateDNSPrefix + path
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DNS_PRIVATEDNS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DNS_PRIVATEDNS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-private-dns")
	privateZoneName := getenv("STACKYARD_AZURE_DNS_PRIVATEDNS_ZONE_NAME", "example.private")
	recordType := getenv("STACKYARD_AZURE_DNS_PRIVATEDNS_RECORD_TYPE", "A")
	relativeRecordSetName := getenv("STACKYARD_AZURE_DNS_PRIVATEDNS_RELATIVE_RECORD_SET_NAME", "www")
	virtualNetworkLinkName := getenv("STACKYARD_AZURE_DNS_PRIVATEDNS_VIRTUAL_NETWORK_LINK_NAME", "vnet-link-1")
	virtualNetworkID := getenv(
		"STACKYARD_AZURE_DNS_PRIVATEDNS_VIRTUAL_NETWORK_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/vnet-private-dns", subscriptionID, resourceGroupName),
	)

	client := newPrivateDNSClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":         subscriptionID,
		"resourceGroupName":      resourceGroupName,
		"privateZoneName":        privateZoneName,
		"recordType":             recordType,
		"relativeRecordSetName":  relativeRecordSetName,
		"virtualNetworkLinkName": virtualNetworkLinkName,
	}

	payloadPrivateZoneCreate := map[string]any{
		"location": "global",
		"tags": map[string]any{
			"env":     "local",
			"service": "private-dns",
		},
	}

	payloadPrivateZoneUpdate := map[string]any{
		"tags": map[string]any{
			"env": "local-updated",
		},
	}

	payloadRecordSetCreate := map[string]any{
		"properties": map[string]any{
			"TTL": 300,
			"ARecords": []any{
				map[string]any{"ipv4Address": "10.0.0.10"},
			},
			"metadata": map[string]any{
				"env": "local",
			},
		},
	}

	payloadRecordSetUpdate := map[string]any{
		"properties": map[string]any{
			"metadata": map[string]any{
				"env": "local-updated",
			},
		},
	}

	payloadVirtualNetworkLinkCreate := map[string]any{
		"location": "global",
		"tags": map[string]any{
			"env": "local",
		},
		"properties": map[string]any{
			"virtualNetwork": map[string]any{
				"id": virtualNetworkID,
			},
			"registrationEnabled": false,
		},
	}

	payloadVirtualNetworkLinkUpdate := map[string]any{
		"tags": map[string]any{
			"env": "local-updated",
		},
		"properties": map[string]any{
			"registrationEnabled": true,
		},
	}

	calls := []privateDNSCall{
		{
			name:     "private_zones_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}?api-version=2018-09-01",
			payload:  payloadPrivateZoneCreate,
			statuses: []int{200, 201, 202},
		},
		{
			name:     "private_zones_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}?api-version=2018-09-01",
			statuses: []int{200},
		},
		{
			name:     "private_zones_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.Network/privateDnsZones?api-version=2018-09-01",
			statuses: []int{200},
		},
		{
			name:     "private_zones_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones?api-version=2018-09-01",
			statuses: []int{200},
		},
		{
			name:     "private_zones_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}?api-version=2018-09-01",
			payload:  payloadPrivateZoneUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "record_sets_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}/{relativeRecordSetName}?api-version=2018-09-01",
			payload:  payloadRecordSetCreate,
			statuses: []int{200, 201},
		},
		{
			name:     "record_sets_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}/{relativeRecordSetName}?api-version=2018-09-01",
			statuses: []int{200},
		},
		{
			name:     "record_sets_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/ALL?api-version=2018-09-01",
			statuses: []int{200},
		},
		{
			name:     "record_sets_list_by_type",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}?api-version=2018-09-01",
			statuses: []int{200},
		},
		{
			name:     "record_sets_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}/{relativeRecordSetName}?api-version=2018-09-01",
			payload:  payloadRecordSetUpdate,
			statuses: []int{200},
		},
		{
			name:     "virtual_network_links_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2018-09-01",
			payload:  payloadVirtualNetworkLinkCreate,
			statuses: []int{200, 201, 202},
		},
		{
			name:     "virtual_network_links_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2018-09-01",
			statuses: []int{200},
		},
		{
			name:     "virtual_network_links_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks?api-version=2018-09-01",
			statuses: []int{200},
		},
		{
			name:     "virtual_network_links_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2018-09-01",
			payload:  payloadVirtualNetworkLinkUpdate,
			statuses: []int{200, 202},
		},
		{
			name:     "record_sets_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}/{relativeRecordSetName}?api-version=2018-09-01",
			statuses: []int{200, 204},
		},
		{
			name:     "virtual_network_links_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2018-09-01",
			statuses: []int{200, 202, 204},
		},
		{
			name:     "private_zones_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}?api-version=2018-09-01",
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
