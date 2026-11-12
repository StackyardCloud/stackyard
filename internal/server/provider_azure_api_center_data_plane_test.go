package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAPICenterDataPlaneRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "list all apis",
			method: http.MethodGet,
			path:   "/azure/apicenter/apis?api-version=2024-02-01-preview",
		},
		{
			name:   "list all environments",
			method: http.MethodGet,
			path:   "/azure/apicenter/environments?api-version=2024-02-01-preview",
		},
		{
			name:   "list workspace apis",
			method: http.MethodGet,
			path:   "/azure/apicenter/workspaces/default/apis?api-version=2024-02-01-preview",
		},
		{
			name:   "get api",
			method: http.MethodGet,
			path:   "/azure/apicenter/workspaces/default/apis/echo-api?api-version=2024-02-01-preview",
		},
		{
			name:   "list versions",
			method: http.MethodGet,
			path:   "/azure/apicenter/workspaces/default/apis/echo-api/versions?api-version=2024-02-01-preview",
		},
		{
			name:   "get version",
			method: http.MethodGet,
			path:   "/azure/apicenter/workspaces/default/apis/echo-api/versions/2023-01-01?api-version=2024-02-01-preview",
		},
		{
			name:   "list definitions",
			method: http.MethodGet,
			path:   "/azure/apicenter/workspaces/default/apis/echo-api/versions/2023-01-01/definitions?api-version=2024-02-01-preview",
		},
		{
			name:   "get definition",
			method: http.MethodGet,
			path:   "/azure/apicenter/workspaces/default/apis/echo-api/versions/2023-01-01/definitions/default?api-version=2024-02-01-preview",
		},
		{
			name:   "export specification",
			method: http.MethodPost,
			path:   "/azure/apicenter/workspaces/default/apis/echo-api/versions/2023-01-01/definitions/default:exportSpecification?api-version=2024-02-01-preview",
			body:   []byte(`{"specification":{"format":"openapi"}}`),
		},
		{
			name:   "get export specification operation status",
			method: http.MethodGet,
			path:   "/azure/apicenter/workspaces/default/apis/echo-api/versions/2023-01-01/definitions/default/operations/00000000-0000-0000-0000-000000000001?api-version=2024-02-01-preview",
		},
		{
			name:   "list deployments",
			method: http.MethodGet,
			path:   "/azure/apicenter/workspaces/default/apis/echo-api/deployments?api-version=2024-02-01-preview",
		},
		{
			name:   "get deployment",
			method: http.MethodGet,
			path:   "/azure/apicenter/workspaces/default/apis/echo-api/deployments/production?api-version=2024-02-01-preview",
		},
		{
			name:   "list workspace environments",
			method: http.MethodGet,
			path:   "/azure/apicenter/workspaces/default/environments?api-version=2024-02-01-preview",
		},
		{
			name:   "get workspace environment",
			method: http.MethodGet,
			path:   "/azure/apicenter/workspaces/default/environments/production?api-version=2024-02-01-preview",
		},
		{
			name:   "alt prefix support",
			method: http.MethodGet,
			path:   "/azure/api-center/workspaces/default/apis?api-version=2024-02-01-preview",
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

func TestAzureAPICenterDataPlaneInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/apicenter/apis?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/apicenter/apis" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureAPICenterDataPlaneUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/apicenter/workspaces/default/apis/echo-api/versions/2023-01-01/definitions/default/custom-preview-route"
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
