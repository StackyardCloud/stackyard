package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureSpeechToTextRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "datasets create",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets?api-version=2024-11-01",
			body:   []byte(`{"displayName":"dataset-a","locale":"en-US","kind":"Acoustic"}`),
		},
		{
			name:   "datasets list",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets?api-version=2024-11-01",
		},
		{
			name:   "datasets get",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets/dataset-a?api-version=2024-11-01",
		},
		{
			name:   "datasets update",
			method: http.MethodPatch,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets/dataset-a?api-version=2024-11-01",
			body:   []byte(`{"description":"updated"}`),
		},
		{
			name:   "datasets delete",
			method: http.MethodDelete,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets/dataset-a?api-version=2024-11-01",
		},
		{
			name:   "datasets files list",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets/dataset-a/files?api-version=2024-11-01",
		},
		{
			name:   "datasets files get",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets/dataset-a/files/file-1?api-version=2024-11-01",
		},
		{
			name:   "datasets upload block",
			method: http.MethodPut,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets/dataset-a/blocks/block-1?api-version=2024-11-01",
			body:   []byte("binary-block"),
		},
		{
			name:   "datasets commit blocks",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets/dataset-a/blocks:commit?api-version=2024-11-01",
			body:   []byte(`{"kind":"Acoustic","links":[]}`),
		},
		{
			name:   "datasets copy",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets/dataset-a:copy?api-version=2024-11-01",
			body:   []byte(`{"targetSubscriptionKey":"local"}`),
		},
		{
			name:   "datasets locales",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets/locales?api-version=2024-11-01",
		},
		{
			name:   "datasets upload",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/datasets/upload?api-version=2024-11-01",
		},
		{
			name:   "endpoints create",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/endpoints?api-version=2024-11-01",
			body:   []byte(`{"displayName":"endpoint-a","locale":"en-US","model":{"self":"https://example/models/model-a"}}`),
		},
		{
			name:   "endpoints list",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/endpoints?api-version=2024-11-01",
		},
		{
			name:   "endpoints get",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/endpoints/endpoint-a?api-version=2024-11-01",
		},
		{
			name:   "endpoints update",
			method: http.MethodPatch,
			path:   "/azure/speechtotext/v3.2-preview.2/endpoints/endpoint-a?api-version=2024-11-01",
			body:   []byte(`{"description":"updated endpoint"}`),
		},
		{
			name:   "endpoints delete",
			method: http.MethodDelete,
			path:   "/azure/speechtotext/v3.2-preview.2/endpoints/endpoint-a?api-version=2024-11-01",
		},
		{
			name:   "endpoints copy",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/endpoints/endpoint-a:copy?api-version=2024-11-01",
			body:   []byte(`{"targetSubscriptionKey":"local"}`),
		},
		{
			name:   "endpoints logs",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/endpoints/endpoint-a/files/logs?api-version=2024-11-01",
		},
		{
			name:   "endpoints audio",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/endpoints/endpoint-a/files/audio?api-version=2024-11-01",
		},
		{
			name:   "evaluations create",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/evaluations?api-version=2024-11-01",
			body:   []byte(`{"displayName":"eval-a","locale":"en-US"}`),
		},
		{
			name:   "evaluations get",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/evaluations/eval-a?api-version=2024-11-01",
		},
		{
			name:   "evaluations update",
			method: http.MethodPatch,
			path:   "/azure/speechtotext/v3.2-preview.2/evaluations/eval-a?api-version=2024-11-01",
			body:   []byte(`{"description":"updated evaluation"}`),
		},
		{
			name:   "evaluations files list",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/evaluations/eval-a/files?api-version=2024-11-01",
		},
		{
			name:   "models create",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/models?api-version=2024-11-01",
			body:   []byte(`{"displayName":"model-a","locale":"en-US"}`),
		},
		{
			name:   "models list",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/models?api-version=2024-11-01",
		},
		{
			name:   "models base list",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/models/base?api-version=2024-11-01",
		},
		{
			name:   "models base get",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/models/base/en-US?api-version=2024-11-01",
		},
		{
			name:   "models authorize copy",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/models:authorizecopy?api-version=2024-11-01",
			body:   []byte(`{"modelId":"model-a"}`),
		},
		{
			name:   "models skus",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/models/skus?api-version=2024-11-01",
		},
		{
			name:   "models manifest",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/models/model-a/manifest?api-version=2024-11-01",
		},
		{
			name:   "projects create",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/projects?api-version=2024-11-01",
			body:   []byte(`{"displayName":"project-a","locale":"en-US"}`),
		},
		{
			name:   "projects list",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/projects?api-version=2024-11-01",
		},
		{
			name:   "projects list models",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/projects/project-a/models?api-version=2024-11-01",
		},
		{
			name:   "projects copy",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/projects/project-a:copy?api-version=2024-11-01",
			body:   []byte(`{"targetSubscriptionKey":"local"}`),
		},
		{
			name:   "operations get",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/operations/op-123?api-version=2024-11-01",
		},
		{
			name:   "service health",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/healthstatus?api-version=2024-11-01",
		},
		{
			name:   "transcriptions create",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/transcriptions?api-version=2024-11-01",
			body:   []byte(`{"displayName":"transcription-a","locale":"en-US"}`),
		},
		{
			name:   "transcriptions list",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/transcriptions?api-version=2024-11-01",
		},
		{
			name:   "transcriptions file get",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/transcriptions/transcription-a/files/file-1?api-version=2024-11-01",
		},
		{
			name:   "webhooks create",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/webhooks?api-version=2024-11-01",
			body:   []byte(`{"displayName":"webhook-a","webUrl":"https://example.com/hook"}`),
		},
		{
			name:   "webhooks list",
			method: http.MethodGet,
			path:   "/azure/speechtotext/v3.2-preview.2/webhooks?api-version=2024-11-01",
		},
		{
			name:   "webhooks ping",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/webhooks/webhook-a/ping?api-version=2024-11-01",
		},
		{
			name:   "webhooks test",
			method: http.MethodPost,
			path:   "/azure/speechtotext/v3.2-preview.2/webhooks/webhook-a/test?api-version=2024-11-01",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := map[string]string{
				"Authorization": "SharedKey devstoreaccount1:signature",
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

func TestAzureSpeechToTextInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/speechtotext/v3.2-preview.2/transcriptions?api-version="
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error payload, got %#v", payload)
	}
	if payload["provider"] != providerAzure || payload["path"] != "/azure/speechtotext/v3.2-preview.2/transcriptions" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureSpeechToTextUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/speechtotext/v3.2-preview.2/unknown-resource"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown speech-to-text nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
