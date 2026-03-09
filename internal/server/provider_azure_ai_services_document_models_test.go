package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAIServicesDocumentModelsRoutesReturnNotImplemented(t *testing.T) {
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
		name        string
		method      string
		path        string
		contentType string
		body        []byte
	}{
		{
			name:   "analyze batch documents",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels/custom-model:analyzeBatch?api-version=2024-11-30",
			body:   []byte(`{"azureBlobSource":{"containerUrl":"https://example.blob.core.windows.net/source"},"resultContainerUrl":"https://example.blob.core.windows.net/results"}`),
		},
		{
			name:   "analyze document",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels/prebuilt-layout:analyze?_overload=analyzeDocument&api-version=2024-11-30",
			body:   []byte(`{"urlSource":"https://example.com/invoice.pdf"}`),
		},
		{
			name:        "analyze document from stream",
			method:      http.MethodPost,
			path:        "/azure/aiservices/documentintelligence/documentModels/prebuilt-layout:analyze?api-version=2024-11-30",
			contentType: "application/pdf",
			body:        []byte("%PDF-1.7 stream-content"),
		},
		{
			name:   "authorize model copy",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels:authorizeCopy?api-version=2024-11-30",
			body:   []byte(`{"modelId":"target-model"}`),
		},
		{
			name:   "build model",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels:build?api-version=2024-11-30",
			body:   []byte(`{"modelId":"custom-model","buildMode":"template","azureBlobSource":{"containerUrl":"https://example.blob.core.windows.net/source"}}`),
		},
		{
			name:   "compose model",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels:compose?api-version=2024-11-30",
			body:   []byte(`{"modelId":"composed-model","componentModels":[{"modelId":"base-model-a"},{"modelId":"base-model-b"}]}`),
		},
		{
			name:   "copy model to target",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentModels/custom-model:copyTo?api-version=2024-11-30",
			body:   []byte(`{"targetResourceId":"resource-id","targetResourceRegion":"eastus","targetModelId":"target-model","accessToken":"token","expirationDateTime":"2099-01-01T00:00:00Z"}`),
		},
		{
			name:   "delete analyze batch result",
			method: http.MethodDelete,
			path:   "/azure/aiservices/documentintelligence/documentModels/custom-model/analyzeBatchResults/batch-result-1?api-version=2024-11-30",
		},
		{
			name:   "delete analyze result",
			method: http.MethodDelete,
			path:   "/azure/aiservices/documentintelligence/documentModels/custom-model/analyzeResults/result-1?api-version=2024-11-30",
		},
		{
			name:   "delete model",
			method: http.MethodDelete,
			path:   "/azure/aiservices/documentintelligence/documentModels/custom-model?api-version=2024-11-30",
		},
		{
			name:   "get analyze batch result",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/documentModels/custom-model/analyzeBatchResults/batch-result-1?api-version=2024-11-30",
		},
		{
			name:   "get analyze result",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/documentModels/custom-model/analyzeResults/result-1?api-version=2024-11-30",
		},
		{
			name:   "get analyze result figure",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/documentModels/custom-model/analyzeResults/result-1/figures/1.1?api-version=2024-11-30",
		},
		{
			name:   "get analyze result pdf",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/documentModels/custom-model/analyzeResults/result-1/pdf?api-version=2024-11-30",
		},
		{
			name:   "get model",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/documentModels/custom-model?api-version=2024-11-30",
		},
		{
			name:   "list analyze batch results",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/documentModels/custom-model/analyzeBatchResults?api-version=2024-11-30",
		},
		{
			name:   "list models",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/documentModels?api-version=2024-11-30",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := map[string]string{
				"Authorization": "SharedKey devstoreaccount1:signature",
			}
			if strings.TrimSpace(tt.contentType) != "" {
				headers["Content-Type"] = tt.contentType
			} else if tt.body != nil {
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

func TestAzureAIServicesDocumentModelsUnknownNestedRouteReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/aiservices/documentintelligence/documentModels/custom-model/unknown"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown document model nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected not-implemented payload: %#v", payload)
	}
}
