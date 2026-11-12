package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureCustomVoiceRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "list base models",
			method: http.MethodGet,
			path:   "/azure/customvoice/basemodels?api-version=2024-02-01-preview",
		},
		{
			name:   "create consent",
			method: http.MethodPut,
			path:   "/azure/customvoice/consents/consent-a?api-version=2024-02-01-preview",
			body:   []byte(`{"email":"stackyard@example.com","fullName":"Stackyard Local"}`),
		},
		{
			name:   "post consent",
			method: http.MethodPost,
			path:   "/azure/customvoice/consents/consent-a?api-version=2024-02-01-preview",
		},
		{
			name:   "list consents",
			method: http.MethodGet,
			path:   "/azure/customvoice/consents?api-version=2024-02-01-preview",
		},
		{
			name:   "create endpoint",
			method: http.MethodPut,
			path:   "/azure/customvoice/endpoints/endpoint-a?api-version=2024-02-01-preview",
			body:   []byte(`{"projectId":"project-a","modelId":"model-a"}`),
		},
		{
			name:   "resume endpoint",
			method: http.MethodPost,
			path:   "/azure/customvoice/endpoints/endpoint-a:resume?api-version=2024-02-01-preview",
		},
		{
			name:   "suspend endpoint",
			method: http.MethodPost,
			path:   "/azure/customvoice/endpoints/endpoint-a:suspend?api-version=2024-02-01-preview",
		},
		{
			name:   "list models",
			method: http.MethodGet,
			path:   "/azure/customvoice/models?api-version=2024-02-01-preview",
		},
		{
			name:   "create model",
			method: http.MethodPut,
			path:   "/azure/customvoice/models/model-a?api-version=2024-02-01-preview",
			body:   []byte(`{"projectId":"project-a","trainingSetIds":["set-a"]}`),
		},
		{
			name:   "list model recipes",
			method: http.MethodGet,
			path:   "/azure/customvoice/modelrecipes?api-version=2024-02-01-preview",
		},
		{
			name:   "get operation",
			method: http.MethodGet,
			path:   "/azure/customvoice/operations/op-123?api-version=2024-02-01-preview",
		},
		{
			name:   "list personal voices",
			method: http.MethodGet,
			path:   "/azure/customvoice/personalvoices?api-version=2024-02-01-preview",
		},
		{
			name:   "create personal voice",
			method: http.MethodPut,
			path:   "/azure/customvoice/personalvoices/personal-a?api-version=2024-02-01-preview",
			body:   []byte(`{"consentId":"consent-a","projectId":"project-a"}`),
		},
		{
			name:   "post personal voice",
			method: http.MethodPost,
			path:   "/azure/customvoice/personalvoices/personal-a?api-version=2024-02-01-preview",
		},
		{
			name:   "list projects",
			method: http.MethodGet,
			path:   "/azure/customvoice/projects?api-version=2024-02-01-preview",
		},
		{
			name:   "create project",
			method: http.MethodPut,
			path:   "/azure/customvoice/projects/project-a?api-version=2024-02-01-preview",
			body:   []byte(`{"displayName":"stackyard-custom-voice-project","locale":"en-US"}`),
		},
		{
			name:   "list training sets",
			method: http.MethodGet,
			path:   "/azure/customvoice/trainingsets?api-version=2024-02-01-preview",
		},
		{
			name:   "create training set",
			method: http.MethodPut,
			path:   "/azure/customvoice/trainingsets/set-a?api-version=2024-02-01-preview",
			body:   []byte(`{"projectId":"project-a","locale":"en-US"}`),
		},
		{
			name:   "upload training set data",
			method: http.MethodPost,
			path:   "/azure/customvoice/trainingsets/set-a:upload?api-version=2024-02-01-preview",
			body:   []byte(`{"files":[{"name":"audio.wav","url":"https://example.com/audio.wav"}]}`),
		},
		{
			name:   "delete training set",
			method: http.MethodDelete,
			path:   "/azure/customvoice/trainingsets/set-a?api-version=2024-02-01-preview",
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

func TestAzureCustomVoiceInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/customvoice/projects?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/customvoice/projects" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureCustomVoiceUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/customvoice/custom/preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown custom voice nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
