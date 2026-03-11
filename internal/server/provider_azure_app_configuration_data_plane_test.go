package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAppConfigurationDataPlaneRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "check key value",
			method: http.MethodHead,
			path:   "/azure/appconfiguration/kv/Message?api-version=2024-09-01",
		},
		{
			name:   "get key value",
			method: http.MethodGet,
			path:   "/azure/appconfiguration/kv/Message?api-version=2024-09-01",
		},
		{
			name:   "put key value",
			method: http.MethodPut,
			path:   "/azure/appconfiguration/kv/Message?api-version=2024-09-01",
			body:   []byte(`{"key":"Message","value":"hello","label":"prod"}`),
		},
		{
			name:   "check key values",
			method: http.MethodHead,
			path:   "/azure/appconfiguration/kv?api-version=2024-09-01",
		},
		{
			name:   "check keys",
			method: http.MethodHead,
			path:   "/azure/appconfiguration/keys?api-version=2024-09-01",
		},
		{
			name:   "check labels",
			method: http.MethodHead,
			path:   "/azure/appconfiguration/labels?api-version=2024-09-01&$select=name",
		},
		{
			name:   "check revisions",
			method: http.MethodHead,
			path:   "/azure/appconfiguration/revisions?api-version=2024-09-01",
		},
		{
			name:   "create snapshot",
			method: http.MethodPut,
			path:   "/azure/appconfiguration/snapshots/Prod-2022-08-01?api-version=2024-09-01",
			body:   []byte(`{"name":"Prod-2022-08-01","filters":[{"key":"*","label":"prod"}]}`),
		},
		{
			name:   "update snapshot",
			method: http.MethodPatch,
			path:   "/azure/appconfiguration/snapshots/Prod-2022-08-01?api-version=2024-09-01",
			body:   []byte(`{"status":"ready"}`),
		},
		{
			name:   "check snapshot",
			method: http.MethodHead,
			path:   "/azure/appconfiguration/snapshots/Prod-2022-08-01?api-version=2024-09-01",
		},
		{
			name:   "check snapshots",
			method: http.MethodHead,
			path:   "/azure/appconfiguration/snapshots?api-version=2024-09-01",
		},
		{
			name:   "put lock",
			method: http.MethodPut,
			path:   "/azure/appconfiguration/locks/Message?api-version=2024-09-01",
		},
		{
			name:   "delete lock",
			method: http.MethodDelete,
			path:   "/azure/appconfiguration/locks/Message?api-version=2024-09-01",
		},
		{
			name:   "get operation details",
			method: http.MethodGet,
			path:   "/azure/appconfiguration/operations?api-version=2024-09-01&snapshot=Prod-2022-08-01",
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

func TestAzureAppConfigurationDataPlaneInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/appconfiguration/kv/Message?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/appconfiguration/kv/Message" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureAppConfigurationDataPlaneUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/appconfiguration/custom-preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown app configuration data plane route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
