package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAccountManagementRoutesReturnFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{
			name:   "accounts create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai?api-version=2024-10-01",
			body:   []byte(`{"location":"eastus","kind":"OpenAI","sku":{"name":"S0"}}`),
		},
		{
			name:   "check domain availability",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/checkDomainAvailability?api-version=2024-10-01",
			body:   []byte(`{"subdomainName":"stackyard-ai","type":"Microsoft.CognitiveServices/accounts"}`),
		},
		{
			name:   "check sku availability",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/checkSkuAvailability?api-version=2024-10-01",
			body:   []byte(`{"skus":["S0"],"kind":"OpenAI","type":"Microsoft.CognitiveServices/accounts"}`),
		},
		{
			name:   "commitment plans list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/commitmentPlans?api-version=2024-10-01",
		},
		{
			name:   "commitment tiers list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/commitmentTiers?api-version=2024-10-01",
		},
		{
			name:   "defender settings patch",
			method: http.MethodPatch,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/defenderForAISettings/default?api-version=2024-10-01",
			body:   []byte(`{"properties":{"state":"Enabled"}}`),
		},
		{
			name:   "deleted accounts get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/locations/eastus/deletedAccounts/stackyard-ai?api-version=2024-10-01",
		},
		{
			name:   "deployments get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/deployments/gpt4o?api-version=2024-10-01",
		},
		{
			name:   "encryption scopes get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/encryptionScopes/default?api-version=2024-10-01",
		},
		{
			name:   "location based model capacities",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/locations/eastus/locationBasedModelCapacities?api-version=2024-10-01",
		},
		{
			name:   "model capacities list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/locations/eastus/modelCapacities?api-version=2024-10-01",
		},
		{
			name:   "models list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/locations/eastus/models?api-version=2024-10-01",
		},
		{
			name:   "network security perimeter configurations get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/networkSecurityPerimeterConfigurations/default?api-version=2024-10-01",
		},
		{
			name:   "operations list",
			method: http.MethodGet,
			path:   "/azure/providers/Microsoft.CognitiveServices/operations?api-version=2024-10-01",
		},
		{
			name:   "private endpoint connections get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/privateEndpointConnections/pec-1?api-version=2024-10-01",
		},
		{
			name:   "private link resources list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/privateLinkResources?api-version=2024-10-01",
		},
		{
			name:   "rai blocklist items add",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/raiBlocklists/default/addRaiBlocklistItems?api-version=2024-10-01",
			body:   []byte(`{"raiBlocklistItems":[{"name":"item-a","pattern":"foo"}]}`),
		},
		{
			name:   "rai blocklists list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/raiBlocklists?api-version=2024-10-01",
		},
		{
			name:   "rai content filters list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/raiContentFilters?api-version=2024-10-01",
		},
		{
			name:   "rai policies list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/raiPolicies?api-version=2024-10-01",
		},
		{
			name:   "resource skus list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/skus?api-version=2024-10-01",
		},
		{
			name:   "usages list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/locations/eastus/usages?api-version=2024-10-01",
		},
		{
			name:   "calculate model capacity",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/locations/eastus:calculateModelCapacity?api-version=2024-10-01",
			body:   []byte(`{"model":{"name":"gpt-4o-mini"},"sku":{"name":"S0"}}`),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := map[string]string{
				"Authorization":             "SharedKey devstoreaccount1:signature",
				"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
			}
			if tt.body != nil {
				headers["Content-Type"] = "application/json"
			}

			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 for %s %s, got %d body=%s", tt.method, tt.path, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			payload := providerContractJSONMap(t, resp)
			if payload["status"] != "ok" {
				t.Fatalf("expected success payload, got %#v", payload)
			}
			if payload["provider"] != providerAzure {
				t.Fatalf("expected provider azure in payload, got %#v", payload)
			}

			expectedPath := tt.path
			if idx := strings.Index(expectedPath, "?"); idx >= 0 {
				expectedPath = expectedPath[:idx]
			}
			if payload["path"] != expectedPath {
				t.Fatalf("expected path %q in payload, got %#v", expectedPath, payload["path"])
			}
		})
	}
}

func TestAzureAccountManagementInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/providers/Microsoft.CognitiveServices/operations?api-version="
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error payload, got %#v", payload)
	}
	if payload["provider"] != providerAzure || payload["path"] != "/azure/providers/Microsoft.CognitiveServices/operations" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureAccountManagementUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ai/providers/Microsoft.CognitiveServices/accounts/stackyard-ai/unknown"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown account-management nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
