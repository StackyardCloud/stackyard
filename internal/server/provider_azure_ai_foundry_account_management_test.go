package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAIFoundryAccountManagementRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "operations list",
			method: http.MethodGet,
			path:   "/azure/providers/Microsoft.CognitiveServices/operations?api-version=2025-06-01",
		},
		{
			name:   "accounts create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-foundry/providers/Microsoft.CognitiveServices/accounts/foundry-a?api-version=2025-06-01",
			body:   []byte(`{"location":"eastus","kind":"AIServices","sku":{"name":"S0"}}`),
		},
		{
			name:   "accounts list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/accounts?api-version=2025-06-01",
		},
		{
			name:   "check domain availability",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/checkDomainAvailability?api-version=2025-06-01",
			body:   []byte(`{"subdomainName":"stackyard-foundry","type":"Microsoft.CognitiveServices/accounts"}`),
		},
		{
			name:   "check sku availability",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/checkSkuAvailability?api-version=2025-06-01",
			body:   []byte(`{"kind":"AIServices","skus":["S0"],"type":"Microsoft.CognitiveServices/accounts"}`),
		},
		{
			name:   "projects create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-foundry/providers/Microsoft.CognitiveServices/accounts/foundry-a/projects/project-a?api-version=2025-06-01",
			body:   []byte(`{"properties":{"description":"stackyard project"}}`),
		},
		{
			name:   "projects list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-foundry/providers/Microsoft.CognitiveServices/accounts/foundry-a/projects?api-version=2025-06-01",
		},
		{
			name:   "project connection create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-foundry/providers/Microsoft.CognitiveServices/accounts/foundry-a/projects/project-a/connections/conn-a?api-version=2025-06-01",
			body:   []byte(`{"properties":{"category":"AzureResource","target":"https://example"}}`),
		},
		{
			name:   "project connections list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-foundry/providers/Microsoft.CognitiveServices/accounts/foundry-a/projects/project-a/connections?api-version=2025-06-01",
		},
		{
			name:   "account capability hosts list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-foundry/providers/Microsoft.CognitiveServices/accounts/foundry-a/accountCapabilityHosts?api-version=2025-06-01",
		},
		{
			name:   "project capability hosts list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-foundry/providers/Microsoft.CognitiveServices/accounts/foundry-a/projects/project-a/projectCapabilityHosts?api-version=2025-06-01",
		},
		{
			name:   "vector store connection create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-foundry/providers/Microsoft.CognitiveServices/accounts/foundry-a/vectorStoreConnections/vsconn-a?api-version=2025-06-01",
			body:   []byte(`{"properties":{"type":"azureBlob","target":"https://example"}}`),
		},
		{
			name:   "vector store connections list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-foundry/providers/Microsoft.CognitiveServices/accounts/foundry-a/vectorStoreConnections?api-version=2025-06-01",
		},
		{
			name:   "usages list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.CognitiveServices/locations/eastus/usages?api-version=2025-06-01",
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

func TestAzureAIFoundryAccountManagementInvalidAPIVersionReturnsBadRequest(t *testing.T) {
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

func TestAzureAIFoundryAccountManagementUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-foundry/providers/Microsoft.CognitiveServices/accounts/foundry-a/custom-preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown ai-foundry account-management nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
