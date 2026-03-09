package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAIServicesDocumentClassifiersRoutesReturnNotImplemented(t *testing.T) {
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
		status      int
	}{
		{
			name:   "authorize copy",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentClassifiers:authorizeCopy?api-version=2024-11-30",
			body:   []byte(`{"classifierId":"target-classifier"}`),
			status: http.StatusOK,
		},
		{
			name:   "build classifier",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentClassifiers:build?api-version=2024-11-30",
			body:   []byte(`{"classifierId":"invoice-classifier","docTypes":{"invoice":{"azureBlobSource":{"containerUrl":"https://example.blob.core.windows.net/data","prefix":"invoices"}}}}`),
			status: http.StatusAccepted,
		},
		{
			name:   "classify document from url",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentClassifiers/invoice-classifier:analyze?api-version=2024-11-30",
			body:   []byte(`{"urlSource":"https://example.com/invoice.pdf"}`),
			status: http.StatusAccepted,
		},
		{
			name:        "classify document from stream",
			method:      http.MethodPost,
			path:        "/azure/aiservices/documentintelligence/documentClassifiers/invoice-classifier:analyze?api-version=2024-11-30",
			contentType: "application/pdf",
			body:        []byte("%PDF-1.7 stream-content"),
			status:      http.StatusAccepted,
		},
		{
			name:   "copy classifier to target",
			method: http.MethodPost,
			path:   "/azure/aiservices/documentintelligence/documentClassifiers/invoice-classifier:copyTo?api-version=2024-11-30",
			body:   []byte(`{"targetClassifierId":"target-classifier","targetResourceId":"resource-id","targetResourceRegion":"eastus","accessToken":"token","expirationDateTime":"2099-01-01T00:00:00Z"}`),
			status: http.StatusAccepted,
		},
		{
			name:   "delete classifier",
			method: http.MethodDelete,
			path:   "/azure/aiservices/documentintelligence/documentClassifiers/invoice-classifier?api-version=2024-11-30",
			status: http.StatusNoContent,
		},
		{
			name:   "get classifier",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/documentClassifiers/invoice-classifier?api-version=2024-11-30",
			status: http.StatusOK,
		},
		{
			name:   "get classify result",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/documentClassifiers/invoice-classifier/analyzeResults/result-1?api-version=2024-11-30",
			status: http.StatusOK,
		},
		{
			name:   "list classifiers",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/documentClassifiers?api-version=2024-11-30",
			status: http.StatusOK,
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
			if resp.StatusCode != tt.status {
				t.Fatalf("expected %d for %s %s, got %d body=%s", tt.status, tt.method, tt.path, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			if tt.status == http.StatusNoContent {
				if got := len(providerContractBody(t, resp)); got != 0 {
					t.Fatalf("expected empty body for %s %s, got %d bytes", tt.method, tt.path, got)
				}
				return
			}
			payload := providerContractJSONMap(t, resp)
			if payload["provider"] != providerAzure {
				t.Fatalf("expected provider azure in not-implemented payload, got %#v", payload)
			}
			if tt.status == http.StatusAccepted {
				if payload["status"] != "running" {
					t.Fatalf("expected running status payload for async route, got %#v", payload)
				}
				return
			}
			if strings.Contains(tt.path, "authorizeCopy") {
				if payload["accessToken"] == nil {
					t.Fatalf("expected accessToken in authorizeCopy payload, got %#v", payload)
				}
				return
			}

			expectedPath := tt.path
			if idx := strings.Index(expectedPath, "?"); idx >= 0 {
				expectedPath = expectedPath[:idx]
			}
			if strings.Contains(tt.path, "documentClassifiers?") {
				if _, ok := payload["value"]; !ok {
					t.Fatalf("expected value list in list payload, got %#v", payload)
				}
			}
			if strings.Contains(tt.path, "/analyzeResults/") {
				if payload["resultId"] == nil {
					t.Fatalf("expected resultId in classify result payload, got %#v", payload)
				}
			}
			if strings.Contains(tt.path, "documentClassifiers/invoice-classifier?") {
				if payload["classifierId"] != "invoice-classifier" {
					t.Fatalf("expected classifierId invoice-classifier, got %#v", payload["classifierId"])
				}
			}
			if gotPath, ok := payload["path"]; ok && gotPath != expectedPath {
				t.Fatalf("expected path %q in payload when present, got %#v", expectedPath, payload["path"])
			}
		})
	}
}

func TestAzureAIServicesDocumentClassifiersUnsupportedRouteReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/aiservices/documentintelligence/unsupported"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unsupported aiservices route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected not-implemented payload: %#v", payload)
	}
}
