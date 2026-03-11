package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAppConfigurationResourceManagerRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "provider operations list",
			method: http.MethodGet,
			path:   "/azure/providers/Microsoft.AppConfiguration/operations?api-version=2024-06-01",
		},
		{
			name:   "check name availability",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AppConfiguration/checkNameAvailability?api-version=2024-06-01",
			body:   []byte(`{"name":"cfg-store","type":"Microsoft.AppConfiguration/configurationStores"}`),
		},
		{
			name:   "regional check name availability",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AppConfiguration/locations/eastus/checkNameAvailability?api-version=2024-06-01",
			body:   []byte(`{"name":"cfg-store","type":"Microsoft.AppConfiguration/configurationStores"}`),
		},
		{
			name:   "configuration store create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-appcfg/providers/Microsoft.AppConfiguration/configurationStores/cfg-store?api-version=2024-06-01",
			body:   []byte(`{"location":"eastus","sku":{"name":"Standard"}}`),
		},
		{
			name:   "configuration store update",
			method: http.MethodPatch,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-appcfg/providers/Microsoft.AppConfiguration/configurationStores/cfg-store?api-version=2024-06-01",
			body:   []byte(`{"tags":{"env":"dev"}}`),
		},
		{
			name:   "configuration store list keys",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-appcfg/providers/Microsoft.AppConfiguration/configurationStores/cfg-store/listKeys?api-version=2024-06-01",
			body:   []byte(`{}`),
		},
		{
			name:   "configuration store regenerate key",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-appcfg/providers/Microsoft.AppConfiguration/configurationStores/cfg-store/regenerateKey?api-version=2024-06-01",
			body:   []byte(`{"id":"primary"}`),
		},
		{
			name:   "key value create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-appcfg/providers/Microsoft.AppConfiguration/configurationStores/cfg-store/keyValues/featureA?api-version=2024-06-01",
			body:   []byte(`{"properties":{"value":"true"}}`),
		},
		{
			name:   "private endpoint connection create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-appcfg/providers/Microsoft.AppConfiguration/configurationStores/cfg-store/privateEndpointConnections/pec-a?api-version=2024-06-01",
			body:   []byte(`{"properties":{"privateLinkServiceConnectionState":{"status":"Approved","description":"ok"}}}`),
		},
		{
			name:   "private link resources list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-appcfg/providers/Microsoft.AppConfiguration/configurationStores/cfg-store/privateLinkResources?api-version=2024-06-01",
		},
		{
			name:   "replica create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-appcfg/providers/Microsoft.AppConfiguration/configurationStores/cfg-store/replicas/replica-a?api-version=2024-06-01",
			body:   []byte(`{"location":"westus2"}`),
		},
		{
			name:   "snapshot create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-appcfg/providers/Microsoft.AppConfiguration/configurationStores/cfg-store/snapshots/snapshot-a?api-version=2024-06-01",
			body:   []byte(`{"properties":{"filters":[{"key":"*"}]}}`),
		},
		{
			name:   "purge deleted configuration store",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AppConfiguration/locations/eastus/deletedConfigurationStores/cfg-store/purge?api-version=2024-06-01",
			body:   []byte(`{}`),
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

func TestAzureAppConfigurationResourceManagerInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/providers/Microsoft.AppConfiguration/operations?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/providers/Microsoft.AppConfiguration/operations" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureAppConfigurationResourceManagerUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-appcfg/providers/Microsoft.AppConfiguration/configurationStores/cfg-store/custom-preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown app configuration nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
