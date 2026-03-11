package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureContentSafetyRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "analyze image",
			method: http.MethodPost,
			path:   "/azure/contentsafety/image:analyze?api-version=2024-09-01",
			body:   []byte(`{"image":{"url":"https://example.com/safe-image.jpg"},"categories":["Hate","Sexual"]}`),
		},
		{
			name:   "analyze text",
			method: http.MethodPost,
			path:   "/azure/contentsafety/text:analyze?api-version=2024-09-01",
			body:   []byte(`{"text":"hello world","categories":["Hate","Sexual"]}`),
		},
		{
			name:   "detect protected material",
			method: http.MethodPost,
			path:   "/azure/contentsafety/text:detectProtectedMaterial?api-version=2024-09-01",
			body:   []byte(`{"text":"protected content input"}`),
		},
		{
			name:   "shield prompt",
			method: http.MethodPost,
			path:   "/azure/contentsafety/text:shieldPrompt?api-version=2024-09-01",
			body:   []byte(`{"userPrompt":"Summarize this text","documents":["doc-one"]}`),
		},
		{
			name:   "list text blocklists",
			method: http.MethodGet,
			path:   "/azure/contentsafety/text/blocklists?api-version=2024-09-01",
		},
		{
			name:   "get text blocklist",
			method: http.MethodGet,
			path:   "/azure/contentsafety/text/blocklists/local-list?api-version=2024-09-01",
		},
		{
			name:   "create or update text blocklist",
			method: http.MethodPatch,
			path:   "/azure/contentsafety/text/blocklists/local-list?api-version=2024-09-01",
			body:   []byte(`{"description":"stackyard local blocklist"}`),
		},
		{
			name:   "delete text blocklist",
			method: http.MethodDelete,
			path:   "/azure/contentsafety/text/blocklists/local-list?api-version=2024-09-01",
		},
		{
			name:   "add or update text blocklist items",
			method: http.MethodPost,
			path:   "/azure/contentsafety/text/blocklists/local-list:addOrUpdateBlocklistItems?api-version=2024-09-01",
			body:   []byte(`{"blocklistItems":[{"text":"forbidden phrase"}]}`),
		},
		{
			name:   "remove text blocklist items",
			method: http.MethodPost,
			path:   "/azure/contentsafety/text/blocklists/local-list:removeBlocklistItems?api-version=2024-09-01",
			body:   []byte(`{"blocklistItemIds":["item-1"]}`),
		},
		{
			name:   "list text blocklist items",
			method: http.MethodGet,
			path:   "/azure/contentsafety/text/blocklists/local-list/blocklistItems?api-version=2024-09-01",
		},
		{
			name:   "get text blocklist item",
			method: http.MethodGet,
			path:   "/azure/contentsafety/text/blocklists/local-list/blocklistItems/item-1?api-version=2024-09-01",
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

func TestAzureContentSafetyInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/contentsafety/text/blocklists?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/contentsafety/text/blocklists" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureContentSafetyUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/contentsafety/custom/preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown content safety route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
