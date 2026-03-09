package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAIServicesLanguageQuestionAnsweringAuthoringRoutesReturnNotImplemented(t *testing.T) {
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
			name:   "add feedback",
			method: http.MethodPost,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/feedback?api-version=2023-04-01",
			body:   []byte(`{"records":[{"userId":"user-1","question":"What is Stackyard?","answerSpan":{"text":"A local cloud emulator"}}]}`),
		},
		{
			name:   "create project",
			method: http.MethodPatch,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a?api-version=2023-04-01",
			body:   []byte(`{"language":"en","description":"stackyard qa authoring project","multilingualResource":false}`),
		},
		{
			name:   "delete project",
			method: http.MethodDelete,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a?api-version=2023-04-01",
		},
		{
			name:   "deploy project",
			method: http.MethodPut,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/deployments/production?api-version=2023-04-01",
			body:   []byte(`{"trainedModelLabel":"latest","rankingModelVersion":"latest"}`),
		},
		{
			name:   "export project",
			method: http.MethodPost,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/:export?api-version=2023-04-01&format=json&assetKind=qnas",
		},
		{
			name:   "get delete status",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects/deletion-jobs/job-123?api-version=2023-04-01",
		},
		{
			name:   "get deploy status",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/deployments/production/jobs/job-123?api-version=2023-04-01",
		},
		{
			name:   "get export status",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/export/jobs/job-123?api-version=2023-04-01",
		},
		{
			name:   "get import status",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/import/jobs/job-123?api-version=2023-04-01",
		},
		{
			name:   "get project details",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a?api-version=2023-04-01",
		},
		{
			name:   "get qnas",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/qnas?api-version=2023-04-01",
		},
		{
			name:   "get sources",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/sources?api-version=2023-04-01",
		},
		{
			name:   "get synonyms",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/synonyms?api-version=2023-04-01",
		},
		{
			name:   "get update qnas status",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/qnas/jobs/job-123?api-version=2023-04-01",
		},
		{
			name:   "get update sources status",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/sources/jobs/job-123?api-version=2023-04-01",
		},
		{
			name:   "import project",
			method: http.MethodPost,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/:import?api-version=2023-04-01&format=json&assetKind=qnas",
		},
		{
			name:   "list deployments",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/deployments?api-version=2023-04-01",
		},
		{
			name:   "list projects",
			method: http.MethodGet,
			path:   "/azure/language/authoring/query-knowledgebases/projects?api-version=2023-04-01",
		},
		{
			name:   "update qnas",
			method: http.MethodPatch,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/qnas?api-version=2023-04-01",
			body:   []byte(`{"value":[{"op":"add","path":"/qnarecords/-","value":{"id":1,"answer":"A local cloud emulator"}}]}`),
		},
		{
			name:   "update sources",
			method: http.MethodPatch,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/sources?api-version=2023-04-01",
			body:   []byte(`{"value":[{"op":"add","value":{"source":"https://example.com/faq"}}]}`),
		},
		{
			name:   "update synonyms",
			method: http.MethodPut,
			path:   "/azure/language/authoring/query-knowledgebases/projects/proj-a/synonyms?api-version=2023-04-01",
			body:   []byte(`{"synonymsGroups":[{"alterations":["stackyard","localstack alternative"]}]}`),
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

func TestAzureAIServicesLanguageQuestionAnsweringAuthoringUnsupportedNestedPathReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/language/authoring/query-knowledgebases/projects/proj-a/unsupported/segment"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown question-answering-authoring nested path, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected not-implemented payload: %#v", payload)
	}
}
