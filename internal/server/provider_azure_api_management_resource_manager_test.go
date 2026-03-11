package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAPIManagementResourceManagerRoutesReturnFoundationSuccess(t *testing.T) {
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
			path:   "/azure/providers/Microsoft.ApiManagement/operations?api-version=2024-05-01",
		},
		{
			name:   "api management skus list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ApiManagement/skus?api-version=2024-05-01",
		},
		{
			name:   "service apply network configuration updates",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/applynetworkconfigurationupdates?api-version=2024-05-01",
			body:   []byte(`{"location":"eastus"}`),
		},
		{
			name:   "service create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a?api-version=2024-05-01",
			body:   []byte(`{"location":"eastus","sku":{"name":"Developer","capacity":1}}`),
		},
		{
			name:   "api create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/apis/echo-api?api-version=2024-05-01",
			body:   []byte(`{"properties":{"displayName":"Echo API","path":"echo","protocols":["https"]}}`),
		},
		{
			name:   "api operation create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/apis/echo-api/operations/get-echo?api-version=2024-05-01",
			body:   []byte(`{"properties":{"displayName":"Get Echo","method":"GET","urlTemplate":"/echo"}}`),
		},
		{
			name:   "api operation policy put",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/apis/echo-api/operations/get-echo/policies/policy?api-version=2024-05-01",
			body:   []byte(`{"properties":{"format":"rawxml","value":"<policies></policies>"}}`),
		},
		{
			name:   "product create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/products/starter?api-version=2024-05-01",
			body:   []byte(`{"properties":{"displayName":"Starter","subscriptionRequired":false,"state":"published"}}`),
		},
		{
			name:   "user create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/users/user-1?api-version=2024-05-01",
			body:   []byte(`{"properties":{"firstName":"Dev","lastName":"User","email":"dev@example.com"}}`),
		},
		{
			name:   "named value create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/namedValues/kv-endpoint?api-version=2024-05-01",
			body:   []byte(`{"properties":{"displayName":"KeyVaultEndpoint","value":"https://example.vault.azure.net/"}}`),
		},
		{
			name:   "gateway create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/gateways/gw-a?api-version=2024-05-01",
			body:   []byte(`{"properties":{"description":"gateway a","locationData":{"name":"eastus"}}}`),
		},
		{
			name:   "workspace create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/workspaces/ws-a?api-version=2024-05-01",
			body:   []byte(`{"properties":{"title":"workspace a"}}`),
		},
		{
			name:   "workspace api create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/workspaces/ws-a/apis/echo-api?api-version=2024-05-01",
			body:   []byte(`{"properties":{"displayName":"Echo API","path":"echo","protocols":["https"]}}`),
		},
		{
			name:   "workspace product create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/workspaces/ws-a/products/starter?api-version=2024-05-01",
			body:   []byte(`{"properties":{"displayName":"Starter"}}`),
		},
		{
			name:   "workspace subscription create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/workspaces/ws-a/subscriptions/sub-a?api-version=2024-05-01",
			body:   []byte(`{"properties":{"displayName":"sub-a","scope":"/products/starter"}}`),
		},
		{
			name:   "workspace named value create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/workspaces/ws-a/namedValues/kv-endpoint?api-version=2024-05-01",
			body:   []byte(`{"properties":{"displayName":"KeyVaultEndpoint","value":"https://example.vault.azure.net/"}}`),
		},
		{
			name:   "notification recipient email head",
			method: http.MethodHead,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/notifications/NewCommentNotificationMessage/recipientEmails/dev-example-com?api-version=2024-05-01",
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
			if tt.method == http.MethodHead {
				return
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

func TestAzureAPIManagementResourceManagerInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/providers/Microsoft.ApiManagement/operations?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/providers/Microsoft.ApiManagement/operations" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureAPIManagementResourceManagerUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apim/providers/Microsoft.ApiManagement/service/apim-a/custom-preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown apimanagement nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
