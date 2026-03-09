package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAIServicesLanguageQuestionAnsweringRoutesReturnNotImplemented(t *testing.T) {
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
			name:   "get answers",
			method: http.MethodPost,
			path:   "/azure/language/:query-knowledgebases?api-version=2023-04-01&projectName=proj-a&deploymentName=production",
			body: []byte(`{
				"question":"What is Stackyard?",
				"top":3,
				"confidenceScoreThreshold":0.1,
				"includeUnstructuredSources":false
			}`),
		},
		{
			name:   "get answers from text",
			method: http.MethodPost,
			path:   "/azure/language/:query-text?api-version=2023-04-01",
			body: []byte(`{
				"question":"What does Stackyard emulate?",
				"textRecords":[
					{"id":"1","text":"Stackyard emulates cloud APIs for local development."}
				]
			}`),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, map[string]string{
				"Authorization": "SharedKey devstoreaccount1:signature",
				"Content-Type":  "application/json",
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

func TestAzureAIServicesLanguageQuestionAnsweringUnsupportedMethodReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/language/:query-knowledgebases"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unsupported method on question answering route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"azure"`) || !strings.Contains(body, path) {
		t.Fatalf("unexpected payload: %s", body)
	}
}
