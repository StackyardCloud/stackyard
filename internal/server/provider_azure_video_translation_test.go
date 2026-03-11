package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureVideoTranslationRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "create event hub configuration",
			method: http.MethodPut,
			path:   "/azure/videotranslation/configurations/event-hub?api-version=2026-03-01",
			body:   []byte(`{"name":"primary","connectionString":"Endpoint=sb://stackyard"}`),
		},
		{
			name:   "get event hub configuration",
			method: http.MethodGet,
			path:   "/azure/videotranslation/configurations/event-hub?api-version=2026-03-01",
		},
		{
			name:   "ping event hub configuration",
			method: http.MethodPost,
			path:   "/azure/videotranslation/configurations/event-hub:ping?api-version=2026-03-01",
		},
		{
			name:   "create translation",
			method: http.MethodPut,
			path:   "/azure/videotranslation/translations/translation-a?api-version=2026-03-01",
			body:   []byte(`{"sourceLocale":"en-US","targetLocales":["es-ES"]}`),
		},
		{
			name:   "get translation",
			method: http.MethodGet,
			path:   "/azure/videotranslation/translations/translation-a?api-version=2026-03-01",
		},
		{
			name:   "list translations",
			method: http.MethodGet,
			path:   "/azure/videotranslation/translations?api-version=2026-03-01&maxpagesize=25",
		},
		{
			name:   "create translation iteration",
			method: http.MethodPut,
			path:   "/azure/videotranslation/translations/translation-a/iterations/iteration-a?api-version=2026-03-01",
			body:   []byte(`{"inputVideoUrl":"https://example.com/video.mp4"}`),
		},
		{
			name:   "get translation iteration",
			method: http.MethodGet,
			path:   "/azure/videotranslation/translations/translation-a/iterations/iteration-a?api-version=2026-03-01",
		},
		{
			name:   "list translation iterations",
			method: http.MethodGet,
			path:   "/azure/videotranslation/translations/translation-a/iterations?api-version=2026-03-01&top=20",
		},
		{
			name:   "get operation",
			method: http.MethodGet,
			path:   "/azure/videotranslation/operations/operation-a?api-version=2026-03-01",
		},
		{
			name:   "delete translation",
			method: http.MethodDelete,
			path:   "/azure/videotranslation/translations/translation-a?api-version=2026-03-01",
		},
		{
			name:   "delete event hub configuration",
			method: http.MethodDelete,
			path:   "/azure/videotranslation/configurations/event-hub?api-version=2026-03-01",
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

func TestAzureVideoTranslationInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/videotranslation/translations?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/videotranslation/translations" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureVideoTranslationMissingPathParameterReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/videotranslation/translations?api-version=2026-03-01"
	resp := providerContractRequest(t, ts, http.MethodPut, path, []byte(`{"sourceLocale":"en-US"}`), map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing translationId path parameter, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error payload, got %#v", payload)
	}
	if payload["message"] != "translationId path parameter is required" {
		t.Fatalf("expected translationId path parameter error, got %#v", payload)
	}
}

func TestAzureVideoTranslationUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/videotranslation/custom/preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown video-translation nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
