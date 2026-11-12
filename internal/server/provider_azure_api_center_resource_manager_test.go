package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAPICenterResourceManagerRoutesReturnFoundationSuccess(t *testing.T) {
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
			path:   "/azure/providers/Microsoft.ApiCenter/operations?api-version=2024-03-01",
		},
		{
			name:   "service create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso?api-version=2024-03-01",
			body:   []byte(`{"location":"eastus","sku":{"name":"Standard"}}`),
		},
		{
			name:   "service update",
			method: http.MethodPatch,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso?api-version=2024-03-01",
			body:   []byte(`{"tags":{"env":"dev"}}`),
		},
		{
			name:   "service export metadata schema",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso/exportMetadataSchema?api-version=2024-03-01",
			body:   []byte(`{"assignedTo":"api","schema":{"name":"author"}}`),
		},
		{
			name:   "service list by resource group",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services?api-version=2024-03-01",
		},
		{
			name:   "service list by subscription",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ApiCenter/services?api-version=2024-03-01",
		},
		{
			name:   "workspace create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso/workspaces/default?api-version=2024-03-01",
			body:   []byte(`{"properties":{"title":"default"}}`),
		},
		{
			name:   "api create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso/workspaces/default/apis/echo-api?api-version=2024-03-01",
			body:   []byte(`{"properties":{"title":"Echo API"}}`),
		},
		{
			name:   "api version create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso/workspaces/default/apis/echo-api/versions/2023-01-01?api-version=2024-03-01",
			body:   []byte(`{"properties":{"title":"2023-01-01"}}`),
		},
		{
			name:   "api definition create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso/workspaces/default/apis/echo-api/versions/2023-01-01/definitions/openapi?api-version=2024-03-01",
			body:   []byte(`{"properties":{"title":"openapi"}}`),
		},
		{
			name:   "api definition export specification",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso/workspaces/default/apis/echo-api/versions/2023-01-01/definitions/openapi/exportSpecification?api-version=2024-03-01",
			body:   []byte(`{"format":"openapi"}`),
		},
		{
			name:   "api definition import specification",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso/workspaces/default/apis/echo-api/versions/2023-01-01/definitions/openapi/importSpecification?api-version=2024-03-01",
			body:   []byte(`{"value":"{\"openapi\":\"3.0.0\"}"}`),
		},
		{
			name:   "deployment create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso/workspaces/default/apis/echo-api/deployments/production?api-version=2024-03-01",
			body:   []byte(`{"properties":{"title":"production"}}`),
		},
		{
			name:   "environment create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso/workspaces/default/environments/public?api-version=2024-03-01",
			body:   []byte(`{"properties":{"title":"public"}}`),
		},
		{
			name:   "metadata schema create",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso/metadataSchemas/author?api-version=2024-03-01",
			body:   []byte(`{"properties":{"schema":{"type":"object"}}}`),
		},
		{
			name:   "head service",
			method: http.MethodHead,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso?api-version=2024-03-01",
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

func TestAzureAPICenterResourceManagerInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/providers/Microsoft.ApiCenter/operations?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/providers/Microsoft.ApiCenter/operations" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureAPICenterResourceManagerUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-apic/providers/Microsoft.ApiCenter/services/contoso/workspaces/default/custom-preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown api-center nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
