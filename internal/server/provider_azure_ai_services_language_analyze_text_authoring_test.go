package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAIServicesLanguageAnalyzeTextAuthoringRoutesReturnNotImplemented(t *testing.T) {
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
			name:   "get supported languages",
			method: http.MethodGet,
			path:   "/azure/language/authoring/analyze-text/projects/proj-a/supportedLanguages?api-version=2023-04-01",
		},
		{
			name:   "create project",
			method: http.MethodPut,
			path:   "/azure/language/authoring/analyze-text/projects/proj-a?api-version=2023-04-01",
			body:   []byte(`{"projectKind":"customSingleLabelClassification","language":"en","multilingual":false}`),
		},
		{
			name:   "export project",
			method: http.MethodPost,
			path:   "/azure/language/authoring/analyze-text/projects/proj-a:export?api-version=2023-04-01",
			body:   []byte(`{"assetKind":"EntireProject","stringIndexType":"Utf16CodeUnit"}`),
		},
		{
			name:   "import project",
			method: http.MethodPost,
			path:   "/azure/language/authoring/analyze-text/projects/proj-a:import?api-version=2023-04-01",
			body:   []byte(`{"projectFileVersion":"2023-04-01","stringIndexType":"Utf16CodeUnit","metadata":{"projectName":"proj-a"}}`),
		},
		{
			name:   "submit train job",
			method: http.MethodPost,
			path:   "/azure/language/authoring/analyze-text/projects/proj-a/train/jobs?api-version=2023-04-01",
			body:   []byte(`{"modelLabel":"model-v1","trainingMode":"standard"}`),
		},
		{
			name:   "cancel train job",
			method: http.MethodPost,
			path:   "/azure/language/authoring/analyze-text/projects/proj-a/train/jobs/job-123:cancel?api-version=2023-04-01",
		},
		{
			name:   "deploy model",
			method: http.MethodPut,
			path:   "/azure/language/authoring/analyze-text/projects/proj-a/deployments/prod?api-version=2023-04-01",
			body:   []byte(`{"trainedModelLabel":"model-v1","deploymentName":"prod"}`),
		},
		{
			name:   "swap deployments",
			method: http.MethodPost,
			path:   "/azure/language/authoring/analyze-text/projects/proj-a/deployments:swap?api-version=2023-04-01",
			body:   []byte(`{"firstDeploymentName":"staging","secondDeploymentName":"prod"}`),
		},
		{
			name:   "list trained models",
			method: http.MethodGet,
			path:   "/azure/language/authoring/analyze-text/projects/proj-a/models?api-version=2023-04-01",
		},
		{
			name:   "load snapshot",
			method: http.MethodPost,
			path:   "/azure/language/authoring/analyze-text/projects/proj-a/models:loadSnapshot?api-version=2023-04-01",
			body:   []byte(`{"trainedModelLabel":"model-v2","modelId":"source-model-id"}`),
		},
		{
			name:   "create text authoring resource",
			method: http.MethodPut,
			path:   "/azure/language/authoring/analyze-text/resources/resource-a?api-version=2023-04-01",
			body:   []byte(`{"resourceId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.CognitiveServices/accounts/authoring"}`),
		},
		{
			name:   "list text authoring resources",
			method: http.MethodGet,
			path:   "/azure/language/authoring/analyze-text/resources?api-version=2023-04-01",
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

func TestAzureAIServicesLanguageAnalyzeTextAuthoringUnsupportedNestedPathReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/language/authoring/analyze-text/projects/proj-a/unknown/segment"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown analyze-text-authoring nested path, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected not-implemented payload: %#v", payload)
	}
}
