package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAnalysisServicesRoutesReturnFoundationSuccess(t *testing.T) {
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
			path:   "/azure/providers/Microsoft.AnalysisServices/operations?api-version=2017-08-01",
		},
		{
			name:   "check name availability",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AnalysisServices/locations/eastus/checkNameAvailability?api-version=2017-08-01",
			body:   []byte(`{"name":"stackyard-aas","type":"Microsoft.AnalysisServices/servers"}`),
		},
		{
			name:   "servers create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aas/providers/Microsoft.AnalysisServices/servers/aas-a?api-version=2017-08-01",
			body:   []byte(`{"location":"eastus","sku":{"name":"S1","tier":"Standard"}}`),
		},
		{
			name:   "servers get details",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aas/providers/Microsoft.AnalysisServices/servers/aas-a?api-version=2017-08-01",
		},
		{
			name:   "servers list by subscription",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AnalysisServices/servers?api-version=2017-08-01",
		},
		{
			name:   "servers list by resource group",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aas/providers/Microsoft.AnalysisServices/servers?api-version=2017-08-01",
		},
		{
			name:   "servers list skus for existing",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aas/providers/Microsoft.AnalysisServices/servers/aas-a/skus?api-version=2017-08-01",
		},
		{
			name:   "servers list skus for new",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AnalysisServices/skus?api-version=2017-08-01",
		},
		{
			name:   "servers list gateway status",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aas/providers/Microsoft.AnalysisServices/servers/aas-a/listGatewayStatus?api-version=2017-08-01",
		},
		{
			name:   "servers dissociate gateway",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aas/providers/Microsoft.AnalysisServices/servers/aas-a/dissociateGateway?api-version=2017-08-01",
		},
		{
			name:   "servers suspend",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aas/providers/Microsoft.AnalysisServices/servers/aas-a/suspend?api-version=2017-08-01",
		},
		{
			name:   "servers resume",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aas/providers/Microsoft.AnalysisServices/servers/aas-a/resume?api-version=2017-08-01",
		},
		{
			name:   "servers update",
			method: http.MethodPatch,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aas/providers/Microsoft.AnalysisServices/servers/aas-a?api-version=2017-08-01",
			body:   []byte(`{"tags":{"env":"test"}}`),
		},
		{
			name:   "servers delete",
			method: http.MethodDelete,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aas/providers/Microsoft.AnalysisServices/servers/aas-a?api-version=2017-08-01",
		},
		{
			name:   "operation results list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AnalysisServices/locations/eastus/operationresults/op-123?api-version=2017-08-01",
		},
		{
			name:   "operation statuses list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AnalysisServices/locations/eastus/operationstatuses/op-123?api-version=2017-08-01",
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

func TestAzureAnalysisServicesInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/providers/Microsoft.AnalysisServices/operations?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/providers/Microsoft.AnalysisServices/operations" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureAnalysisServicesUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aas/providers/Microsoft.AnalysisServices/servers/aas-a/custom-preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown analysis services nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
