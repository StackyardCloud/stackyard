package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureContentUnderstandingRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "list analyzers",
			method: http.MethodGet,
			path:   "/azure/contentunderstanding/analyzers?api-version=2025-11-01",
		},
		{
			name:   "create or replace analyzer",
			method: http.MethodPut,
			path:   "/azure/contentunderstanding/analyzers/analyzer-a?api-version=2025-11-01",
			body:   []byte(`{"description":"stackyard analyzer","scenario":"generic"}`),
		},
		{
			name:   "update analyzer",
			method: http.MethodPatch,
			path:   "/azure/contentunderstanding/analyzers/analyzer-a?api-version=2025-11-01",
			body:   []byte(`{"description":"updated analyzer"}`),
		},
		{
			name:   "get analyzer",
			method: http.MethodGet,
			path:   "/azure/contentunderstanding/analyzers/analyzer-a?api-version=2025-11-01",
		},
		{
			name:   "delete analyzer",
			method: http.MethodDelete,
			path:   "/azure/contentunderstanding/analyzers/analyzer-a?api-version=2025-11-01",
		},
		{
			name:   "analyze content",
			method: http.MethodPost,
			path:   "/azure/contentunderstanding/analyzers/analyzer-a:analyze?api-version=2025-11-01",
			body:   []byte(`{"documentLocation":"https://example.com/invoice.pdf"}`),
		},
		{
			name:   "analyze binary content",
			method: http.MethodPost,
			path:   "/azure/contentunderstanding/analyzers/analyzer-a:analyzeBinary?api-version=2025-11-01",
			body:   []byte("binary-content"),
		},
		{
			name:   "copy analyzer",
			method: http.MethodPost,
			path:   "/azure/contentunderstanding/analyzers/analyzer-a:copy?api-version=2025-11-01",
			body:   []byte(`{"targetAnalyzerResourceId":"/subscriptions/000/resourceGroups/rg/providers/Microsoft.CognitiveServices/accounts/ai/analyzers/target-a"}`),
		},
		{
			name:   "grant copy authorization",
			method: http.MethodPost,
			path:   "/azure/contentunderstanding/analyzers/analyzer-a:grantCopyAuthorization?api-version=2025-11-01",
			body:   []byte(`{"targetRegion":"eastus","targetAnalyzerId":"target-a"}`),
		},
		{
			name:   "get operation status",
			method: http.MethodGet,
			path:   "/azure/contentunderstanding/analyzers/analyzer-a/operations/op-1?api-version=2025-11-01",
		},
		{
			name:   "get analyzer result",
			method: http.MethodGet,
			path:   "/azure/contentunderstanding/analyzerResults/op-1?api-version=2025-11-01",
		},
		{
			name:   "delete analyzer result",
			method: http.MethodDelete,
			path:   "/azure/contentunderstanding/analyzerResults/op-1?api-version=2025-11-01",
		},
		{
			name:   "get analyzer result file",
			method: http.MethodGet,
			path:   "/azure/contentunderstanding/analyzerResults/op-1/files/file-1?api-version=2025-11-01",
		},
		{
			name:   "get defaults",
			method: http.MethodGet,
			path:   "/azure/contentunderstanding/defaults?api-version=2025-11-01",
		},
		{
			name:   "update defaults",
			method: http.MethodPatch,
			path:   "/azure/contentunderstanding/defaults?api-version=2025-11-01",
			body:   []byte(`{"defaultLocale":"en-US"}`),
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

func TestAzureContentUnderstandingInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/contentunderstanding/analyzers?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/contentunderstanding/analyzers" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureContentUnderstandingUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/contentunderstanding/custom/preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown content understanding route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
