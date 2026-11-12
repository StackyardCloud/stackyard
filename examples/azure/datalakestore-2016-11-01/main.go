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

const dataLakeStoreAPIVersion = "2016-11-01"

type dataLakeStoreClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dataLakeStoreSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dataLakeStoreSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDataLakeStoreClient(endpoint, account, subscriptionKey string) *dataLakeStoreClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"datalakestore-2016-11-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&dataLakeStoreSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &dataLakeStoreClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataLakeStoreClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DATALAKESTORE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DATALAKESTORE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-datalakestore")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus2")
	accountName := getenv("STACKYARD_AZURE_DATALAKESTORE_ACCOUNT_NAME", "contosoadls")
	firewallRuleName := getenv("STACKYARD_AZURE_DATALAKESTORE_FIREWALL_RULE_NAME", "rule-example")
	trustedIDProviderName := getenv("STACKYARD_AZURE_DATALAKESTORE_TRUSTED_ID_PROVIDER_NAME", "trusted-id-provider-example")
	virtualNetworkRuleName := getenv("STACKYARD_AZURE_DATALAKESTORE_VNET_RULE_NAME", "vnet-rule-example")
	subnetID := getenv(
		"STACKYARD_AZURE_DATALAKESTORE_SUBNET_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/example-vnet/subnets/example-subnet",
			subscriptionID,
			resourceGroup,
		),
	)

	client := newDataLakeStoreClient(endpoint, account, subscriptionKey)

	accountBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataLakeStore/accounts/%s",
		subscriptionID,
		resourceGroup,
		accountName,
	)

	firewallRulePath := accountBase + "/firewallRules/" + firewallRuleName + "?api-version=" + dataLakeStoreAPIVersion
	trustedIDProviderPath := accountBase + "/trustedIdProviders/" + trustedIDProviderName + "?api-version=" + dataLakeStoreAPIVersion
	virtualNetworkRulePath := accountBase + "/virtualNetworkRules/" + virtualNetworkRuleName + "?api-version=" + dataLakeStoreAPIVersion

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
			path:     "/azure/providers/Microsoft.DataLakeStore/operations?api-version=" + dataLakeStoreAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/providers/Microsoft.DataLakeStore/locations/%s/checkNameAvailability?api-version=%s",
				subscriptionID,
				location,
				dataLakeStoreAPIVersion,
			),
			payload: map[string]any{
				"name": accountName,
				"type": "Microsoft.DataLakeStore/accounts",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateAccount",
			method: http.MethodPut,
			path:   accountBase + "?api-version=" + dataLakeStoreAPIVersion,
			payload: map[string]any{
				"location": location,
				"tags": map[string]any{
					"environment": "local",
					"service":     "datalakestore",
				},
				"properties": map[string]any{
					"defaultGroup":           "data-scientists",
					"firewallState":          "Enabled",
					"firewallAllowAzureIps":  "Enabled",
					"trustedIdProviderState": "Enabled",
					"newTier":                "Consumption",
					"firewallRules": []any{
						map[string]any{
							"name": firewallRuleName,
							"properties": map[string]any{
								"startIpAddress": "1.1.1.1",
								"endIpAddress":   "2.2.2.2",
							},
						},
					},
					"trustedIdProviders": []any{
						map[string]any{
							"name": trustedIDProviderName,
							"properties": map[string]any{
								"idProvider": "https://sts.windows.net/ea9ec534-a3e3-4e45-ad36-3afc5bb291c1",
							},
						},
					},
					"virtualNetworkRules": []any{
						map[string]any{
							"name": virtualNetworkRuleName,
							"properties": map[string]any{
								"subnetId": subnetID,
							},
						},
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetAccount",
			method:   http.MethodGet,
			path:     accountBase + "?api-version=" + dataLakeStoreAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateAccount",
			method: http.MethodPatch,
			path:   accountBase + "?api-version=" + dataLakeStoreAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"environment": "local-updated",
					"service":     "datalakestore",
				},
				"properties": map[string]any{
					"defaultGroup": "analysts",
					"newTier":      "Consumption",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "ListAccounts",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataLakeStore/accounts?api-version=%s", subscriptionID, dataLakeStoreAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAccountsByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataLakeStore/accounts?api-version=%s", subscriptionID, resourceGroup, dataLakeStoreAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "EnableKeyVault",
			method:   http.MethodPost,
			path:     accountBase + "/enableKeyVault?api-version=" + dataLakeStoreAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "CreateOrUpdateFirewallRule",
			method: http.MethodPut,
			path:   firewallRulePath,
			payload: map[string]any{
				"properties": map[string]any{
					"startIpAddress": "1.1.1.1",
					"endIpAddress":   "2.2.2.2",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetFirewallRule",
			method:   http.MethodGet,
			path:     firewallRulePath,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateFirewallRule",
			method: http.MethodPatch,
			path:   firewallRulePath,
			payload: map[string]any{
				"properties": map[string]any{
					"startIpAddress": "1.1.1.2",
					"endIpAddress":   "2.2.2.3",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFirewallRulesByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/firewallRules?api-version=" + dataLakeStoreAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateTrustedIDProvider",
			method: http.MethodPut,
			path:   trustedIDProviderPath,
			payload: map[string]any{
				"properties": map[string]any{
					"idProvider": "https://sts.windows.net/ea9ec534-a3e3-4e45-ad36-3afc5bb291c1",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetTrustedIDProvider",
			method:   http.MethodGet,
			path:     trustedIDProviderPath,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateTrustedIDProvider",
			method: http.MethodPatch,
			path:   trustedIDProviderPath,
			payload: map[string]any{
				"properties": map[string]any{
					"idProvider": "https://sts.windows.net/ea9ec534-a3e3-4e45-ad36-3afc5bb291c2",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListTrustedIDProvidersByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/trustedIdProviders?api-version=" + dataLakeStoreAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateVirtualNetworkRule",
			method: http.MethodPut,
			path:   virtualNetworkRulePath,
			payload: map[string]any{
				"properties": map[string]any{
					"subnetId": subnetID,
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetVirtualNetworkRule",
			method:   http.MethodGet,
			path:     virtualNetworkRulePath,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateVirtualNetworkRule",
			method: http.MethodPatch,
			path:   virtualNetworkRulePath,
			payload: map[string]any{
				"properties": map[string]any{
					"subnetId": subnetID,
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListVirtualNetworkRulesByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/virtualNetworkRules?api-version=" + dataLakeStoreAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetLocationCapability",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataLakeStore/locations/%s/capability?api-version=%s", subscriptionID, location, dataLakeStoreAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:     "GetLocationUsage",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataLakeStore/locations/%s/usages?api-version=%s", subscriptionID, location, dataLakeStoreAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteVirtualNetworkRule",
			method:   http.MethodDelete,
			path:     virtualNetworkRulePath,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteTrustedIDProvider",
			method:   http.MethodDelete,
			path:     trustedIDProviderPath,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteFirewallRule",
			method:   http.MethodDelete,
			path:     firewallRulePath,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteAccount",
			method:   http.MethodDelete,
			path:     accountBase + "?api-version=" + dataLakeStoreAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
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

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	if len(expectedStatuses) == 0 {
		if status < 200 || status >= 300 {
			return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
		}
		return nil
	}
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func printJSON(name string, status int, payload map[string]any) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
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
