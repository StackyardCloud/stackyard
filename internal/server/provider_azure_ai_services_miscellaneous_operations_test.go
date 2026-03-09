package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAIServicesMiscellaneousOperationsRoutesReturnNotImplemented(t *testing.T) {
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
	}{
		{
			name:   "get operation details",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/operations/op-123?_overload=getOperation&api-version=2024-11-30",
		},
		{
			name:   "get document model build operation",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/operations/op-123?api-version=2024-11-30",
		},
		{
			name:   "get document model compose operation",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/operations/op-123?_overload=getDocumentModelComposeOperation&api-version=2024-11-30",
		},
		{
			name:   "get document model copy operation",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/operations/op-123?_overload=getDocumentModelCopyToOperation&api-version=2024-11-30",
		},
		{
			name:   "get document classifier build operation",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/operations/op-123?_overload=getDocumentClassifierBuildOperation&api-version=2024-11-30",
		},
		{
			name:   "get document classifier copy operation",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/operations/op-123?_overload=getDocumentClassifierCopyToOperation&api-version=2024-11-30",
		},
		{
			name:   "get resource details",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/info?api-version=2024-11-30",
		},
		{
			name:   "list operations",
			method: http.MethodGet,
			path:   "/azure/aiservices/documentintelligence/operations?api-version=2024-11-30",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := providerContractRequest(t, ts, tt.method, tt.path, nil, map[string]string{
				"Authorization": "SharedKey devstoreaccount1:signature",
			})
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

func TestAzureAIServicesMiscellaneousOperationsUnsupportedNestedPathReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/aiservices/documentintelligence/operations/op-123/status"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unsupported misc path, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected not-implemented payload: %#v", payload)
	}
}
