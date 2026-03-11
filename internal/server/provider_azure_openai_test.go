package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureOpenAIRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "batch create",
			method: http.MethodPost,
			path:   "/azure/openai/batches?api-version=2024-10-21",
			body:   []byte(`{"input_file_id":"file-abc","endpoint":"/chat/completions","completion_window":"24h"}`),
		},
		{
			name:   "batch list",
			method: http.MethodGet,
			path:   "/azure/openai/batches?api-version=2024-10-21",
		},
		{
			name:   "batch get",
			method: http.MethodGet,
			path:   "/azure/openai/batches/batch-123?api-version=2024-10-21",
		},
		{
			name:   "batch cancel",
			method: http.MethodPost,
			path:   "/azure/openai/batches/batch-123/cancel?api-version=2024-10-21",
		},
		{
			name:   "file upload",
			method: http.MethodPost,
			path:   "/azure/openai/files?api-version=2024-10-21",
			body:   []byte(`{"purpose":"assistants"}`),
		},
		{
			name:   "file list",
			method: http.MethodGet,
			path:   "/azure/openai/files?api-version=2024-10-21",
		},
		{
			name:   "file get",
			method: http.MethodGet,
			path:   "/azure/openai/files/file-abc?api-version=2024-10-21",
		},
		{
			name:   "file delete",
			method: http.MethodDelete,
			path:   "/azure/openai/files/file-abc?api-version=2024-10-21",
		},
		{
			name:   "file content",
			method: http.MethodGet,
			path:   "/azure/openai/files/file-abc/content?api-version=2024-10-21",
		},
		{
			name:   "file import",
			method: http.MethodPost,
			path:   "/azure/openai/files/import?api-version=2024-10-21",
			body:   []byte(`{"purpose":"assistants","source":{"type":"azure_storage","container_url":"https://example.blob.core.windows.net/data"}}`),
		},
		{
			name:   "fine tuning create job",
			method: http.MethodPost,
			path:   "/azure/openai/fine_tuning/jobs?api-version=2024-10-21",
			body:   []byte(`{"training_file":"file-abc","model":"gpt-4o-mini"}`),
		},
		{
			name:   "fine tuning list jobs",
			method: http.MethodGet,
			path:   "/azure/openai/fine_tuning/jobs?api-version=2024-10-21",
		},
		{
			name:   "fine tuning get job",
			method: http.MethodGet,
			path:   "/azure/openai/fine_tuning/jobs/ftjob-123?api-version=2024-10-21",
		},
		{
			name:   "fine tuning cancel job",
			method: http.MethodPost,
			path:   "/azure/openai/fine_tuning/jobs/ftjob-123/cancel?api-version=2024-10-21",
		},
		{
			name:   "fine tuning list checkpoints",
			method: http.MethodGet,
			path:   "/azure/openai/fine_tuning/jobs/ftjob-123/checkpoints?api-version=2024-10-21",
		},
		{
			name:   "fine tuning list events",
			method: http.MethodGet,
			path:   "/azure/openai/fine_tuning/jobs/ftjob-123/events?api-version=2024-10-21",
		},
		{
			name:   "fine tuning delete job",
			method: http.MethodDelete,
			path:   "/azure/openai/fine_tuning/jobs/ftjob-123?api-version=2024-10-21",
		},
		{
			name:   "models list",
			method: http.MethodGet,
			path:   "/azure/openai/models?api-version=2024-10-21",
		},
		{
			name:   "models get",
			method: http.MethodGet,
			path:   "/azure/openai/models/gpt-4o-mini?api-version=2024-10-21",
		},
		{
			name:   "uploads cancel",
			method: http.MethodPost,
			path:   "/azure/openai/uploads/upload-123/cancel?api-version=2024-10-21",
		},
		{
			name:   "uploads complete",
			method: http.MethodPost,
			path:   "/azure/openai/uploads/upload-123/complete?api-version=2024-10-21",
			body:   []byte(`{"part_ids":["part-1","part-2"]}`),
		},
		{
			name:   "uploads add part",
			method: http.MethodPost,
			path:   "/azure/openai/uploads/upload-123/parts?api-version=2024-10-21",
			body:   []byte(`{"content":"cGFydA=="}`),
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

func TestAzureOpenAIInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/openai/models?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/openai/models" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureOpenAIUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/openai/unknown-resource"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown openai nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
