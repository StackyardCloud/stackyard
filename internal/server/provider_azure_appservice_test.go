package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAppServiceRoutesReturnFoundationSuccess(t *testing.T) {
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
			path:   "/azure/providers/Microsoft.Web/operations?api-version=2025-05-01",
		},
		{
			name:   "check name availability",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Web/checknameavailability?api-version=2025-05-01",
			body:   []byte(`{"name":"stackyard-app","type":"Site"}`),
		},
		{
			name:   "regional check name availability",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Web/checknameavailability?api-version=2025-05-01&isFqdn=true",
			body:   []byte(`{"name":"stackyard-app","type":"Microsoft.Web/sites"}`),
		},
		{
			name:   "app service plan create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/serverfarms/plan-a?api-version=2025-05-01",
			body:   []byte(`{"location":"eastus","sku":{"name":"F1","tier":"Free","capacity":1}}`),
		},
		{
			name:   "app service plan get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/serverfarms/plan-a?api-version=2025-05-01",
		},
		{
			name:   "app service plan list by resource group",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/serverfarms?api-version=2025-05-01",
		},
		{
			name:   "certificate get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/certificates/cert-a?api-version=2025-05-01",
		},
		{
			name:   "app service environment get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/hostingEnvironments/ase-a?api-version=2025-05-01",
		},
		{
			name:   "kube environment get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/kubeEnvironments/kube-a?api-version=2025-05-01",
		},
		{
			name:   "web app create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/sites/site-a?api-version=2025-05-01",
			body:   []byte(`{"location":"eastus","properties":{"serverFarmId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/serverfarms/plan-a","httpsOnly":true}}`),
		},
		{
			name:   "web app get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/sites/site-a?api-version=2025-05-01",
		},
		{
			name:   "web app config update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/sites/site-a/config/web?api-version=2025-05-01",
			body:   []byte(`{"properties":{"alwaysOn":true,"http20Enabled":true}}`),
		},
		{
			name:   "web app config get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/sites/site-a/config/web?api-version=2025-05-01",
		},
		{
			name:   "web app host name binding get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/sites/site-a/hostNameBindings/www.example.com?api-version=2025-05-01",
		},
		{
			name:   "site certificates list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/sites/site-a/siteCertificates?api-version=2025-05-01",
		},
		{
			name:   "static site create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/staticSites/static-a?api-version=2025-05-01",
			body:   []byte(`{"location":"eastus2","properties":{"repositoryURL":"https://github.com/example/static-a","branch":"main"}}`),
		},
		{
			name:   "static site get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/staticSites/static-a?api-version=2025-05-01",
		},
		{
			name:   "static site custom domain get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/staticSites/static-a/customDomains/www.example.com?api-version=2025-05-01",
		},
		{
			name:   "workflow get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/sites/site-a/hostruntime/runtime/webhooks/workflow/api/management/workflows/workflow-a?api-version=2025-05-01",
		},
		{
			name:   "workflow run cancel",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/sites/site-a/hostruntime/runtime/webhooks/workflow/api/management/workflows/workflow-a/runs/run-a/cancel?api-version=2025-05-01",
		},
		{
			name:   "workflow regenerate access key",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/sites/site-a/hostruntime/runtime/webhooks/workflow/api/management/workflows/workflow-a/regenerateAccessKey?api-version=2025-05-01",
			body:   []byte(`{"keyType":"Primary"}`),
		},
		{
			name:   "head web app route allowed",
			method: http.MethodHead,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/sites/site-a?api-version=2025-05-01",
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
				return // HEAD responses carry no body
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

func TestAzureAppServiceInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/providers/Microsoft.Web/operations?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/providers/Microsoft.Web/operations" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureAppServiceUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.Web/sites/site-a/custom-preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown appservice nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
