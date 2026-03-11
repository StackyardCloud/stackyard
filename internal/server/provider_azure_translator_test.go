package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureTranslatorRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "translate text",
			method: http.MethodPost,
			path:   "/azure/translator/translate?api-version=3.0&from=en&to=es",
			body:   []byte(`[{"Text":"hello world"}]`),
		},
		{
			name:   "detect language",
			method: http.MethodPost,
			path:   "/azure/translator/detect?api-version=3.0",
			body:   []byte(`[{"Text":"hola mundo"}]`),
		},
		{
			name:   "break sentence",
			method: http.MethodPost,
			path:   "/azure/translator/breaksentence?api-version=3.0&language=en",
			body:   []byte(`[{"Text":"hello world. this is stackyard."}]`),
		},
		{
			name:   "dictionary lookup",
			method: http.MethodPost,
			path:   "/azure/translator/dictionary/lookup?api-version=3.0&from=en&to=es",
			body:   []byte(`[{"Text":"work"}]`),
		},
		{
			name:   "dictionary examples",
			method: http.MethodPost,
			path:   "/azure/translator/dictionary/examples?api-version=3.0&from=en&to=es",
			body:   []byte(`[{"Text":"work","Translation":"trabajo"}]`),
		},
		{
			name:   "languages",
			method: http.MethodGet,
			path:   "/azure/translator/languages?api-version=3.0&scope=translation,transliteration,dictionary",
		},
		{
			name:   "transliterate",
			method: http.MethodPost,
			path:   "/azure/translator/transliterate?api-version=3.0&language=ja&fromScript=Jpan&toScript=Latn",
			body:   []byte(`[{"Text":"こんにちは"}]`),
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
				headers["Content-Type"] = "application/json; charset=UTF-8"
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

func TestAzureTranslatorInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/translator/translate?api-version="
	resp := providerContractRequest(t, ts, http.MethodPost, path, []byte(`[{"Text":"hello"}]`), map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json; charset=UTF-8",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error payload, got %#v", payload)
	}
	if payload["provider"] != providerAzure || payload["path"] != "/azure/translator/translate" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureTranslatorMissingRequiredQueryReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/translator/transliterate?api-version=3.0&language=ja&fromScript=Jpan"
	resp := providerContractRequest(t, ts, http.MethodPost, path, []byte(`[{"Text":"こんにちは"}]`), map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json; charset=UTF-8",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing transliterate query parameter, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error payload, got %#v", payload)
	}
	if payload["message"] != "toScript query parameter is required" {
		t.Fatalf("expected toScript query parameter error, got %#v", payload)
	}
}

func TestAzureTranslatorUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/translator/custom/preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown translator nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
