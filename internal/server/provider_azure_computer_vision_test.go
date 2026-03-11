package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureComputerVisionRoutesReturnFoundationSuccess(t *testing.T) {
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
			method: http.MethodPut,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/datasets/dataset-a?api-version=2023-04-01-preview",
			body:   []byte(`{"displayName":"dataset-a","description":"stackyard dataset"}`),
		},
		{
			name:   "datasets delete",
			method: http.MethodDelete,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/datasets/dataset-a?api-version=2023-04-01-preview",
		},
		{
			name:   "datasets get",
			method: http.MethodGet,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/datasets/dataset-a?api-version=2023-04-01-preview",
		},
		{
			name:   "datasets list",
			method: http.MethodGet,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/datasets?api-version=2023-04-01-preview",
		},
		{
			name:   "datasets update",
			method: http.MethodPatch,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/datasets/dataset-a?api-version=2023-04-01-preview",
			body:   []byte(`{"description":"updated dataset description"}`),
		},
		{
			name:   "image analysis analyze image",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imageanalysis:analyze?api-version=2023-04-01-preview",
			body:   []byte(`{"url":"https://example.com/image.jpg","features":["caption"]}`),
		},
		{
			name:   "image analysis analyze stream",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imageanalysis:analyze?api-version=2023-04-01-preview",
			body:   []byte("stream-content"),
		},
		{
			name:   "image analysis segment image",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imageanalysis:segment?api-version=2023-04-01-preview",
			body:   []byte(`{"url":"https://example.com/image.jpg","mode":"foregroundMatting"}`),
		},
		{
			name:   "image analysis segment stream",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imageanalysis:segment?api-version=2023-04-01-preview",
			body:   []byte("stream-content"),
		},
		{
			name:   "image composition rectify image",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imagecomposition:rectify?api-version=2023-04-01-preview",
			body:   []byte(`{"url":"https://example.com/image.jpg"}`),
		},
		{
			name:   "image composition stitch images",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imagecomposition:stitch?api-version=2023-04-01-preview",
			body:   []byte(`{"images":["https://example.com/1.jpg","https://example.com/2.jpg"]}`),
		},
		{
			name:   "image retrieval vectorize image",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imageretrieval:vectorizeimage?api-version=2023-04-01-preview",
			body:   []byte(`{"url":"https://example.com/image.jpg"}`),
		},
		{
			name:   "image retrieval vectorize stream",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imageretrieval:vectorizestream?api-version=2023-04-01-preview",
			body:   []byte("stream-content"),
		},
		{
			name:   "image retrieval vectorize text",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/imageretrieval:vectorizetext?api-version=2023-04-01-preview",
			body:   []byte(`{"text":"stackyard vision search"}`),
		},
		{
			name:   "model evaluations create",
			method: http.MethodPut,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/modelevaluations/eval-a?api-version=2023-04-01-preview",
			body:   []byte(`{"dataset":"dataset-a","metrics":["precision","recall"]}`),
		},
		{
			name:   "model evaluations delete",
			method: http.MethodDelete,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/modelevaluations/eval-a?api-version=2023-04-01-preview",
		},
		{
			name:   "model evaluations get",
			method: http.MethodGet,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/modelevaluations/eval-a?api-version=2023-04-01-preview",
		},
		{
			name:   "model evaluations list",
			method: http.MethodGet,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/modelevaluations?api-version=2023-04-01-preview",
		},
		{
			name:   "models cancel",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/models/model-a:cancel?api-version=2023-04-01-preview",
		},
		{
			name:   "models create",
			method: http.MethodPut,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/models/model-a?api-version=2023-04-01-preview",
			body:   []byte(`{"kind":"productRecognition","description":"demo model"}`),
		},
		{
			name:   "models delete",
			method: http.MethodDelete,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/models/model-a?api-version=2023-04-01-preview",
		},
		{
			name:   "models get",
			method: http.MethodGet,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/models/model-a?api-version=2023-04-01-preview",
		},
		{
			name:   "models list",
			method: http.MethodGet,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/models?api-version=2023-04-01-preview",
		},
		{
			name:   "planogram compliance match",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/planogramcompliance:match?api-version=2023-04-01-preview",
			body:   []byte(`{"planogram":{"items":[]},"observed":{"items":[]}}`),
		},
		{
			name:   "product recognition create run",
			method: http.MethodPost,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/productrecognition/runs?api-version=2023-04-01-preview",
			body:   []byte(`{"modelName":"model-a","image":"https://example.com/shelf.jpg"}`),
		},
		{
			name:   "product recognition delete run",
			method: http.MethodDelete,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/productrecognition/runs/run-a?api-version=2023-04-01-preview",
		},
		{
			name:   "product recognition get run",
			method: http.MethodGet,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/productrecognition/runs/run-a?api-version=2023-04-01-preview",
		},
		{
			name:   "product recognition list runs",
			method: http.MethodGet,
			path:   "/azure/computervision/v4.0-preview/2023-04-01/productrecognition/runs?api-version=2023-04-01-preview",
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

func TestAzureComputerVisionInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/computervision/v4.0-preview/2023-04-01/models?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/computervision/v4.0-preview/2023-04-01/models" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureComputerVisionUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/computervision/v4.0-preview/2023-04-01/unknown-resource"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown computer-vision nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
