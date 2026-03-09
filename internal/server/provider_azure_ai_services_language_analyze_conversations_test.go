package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAIServicesLanguageAnalyzeConversationsRoutesReturnNotImplemented(t *testing.T) {
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
			name:   "analyze conversations",
			method: http.MethodPost,
			path:   "/azure/language/:analyze-conversations?api-version=2024-11-01&showStats=true",
			body: []byte(`{
				"kind":"Conversation",
				"analysisInput":{
					"conversationItem":{
						"id":"1",
						"participantId":"user-1",
						"modality":"text",
						"language":"en",
						"text":"I need help with my billing issue."
					}
				},
				"parameters":{
					"projectName":"billing-project",
					"deploymentName":"production"
				}
			}`),
		},
		{
			name:   "submit analysis job",
			method: http.MethodPost,
			path:   "/azure/language/analyze-conversations/jobs?api-version=2024-11-01",
			body: []byte(`{
				"displayName":"stackyard-analyze-conversations-job",
				"analysisInput":{
					"conversations":[
						{
							"id":"conversation-1",
							"language":"en",
							"conversationItems":[
								{
									"id":"1",
									"participantId":"user-1",
									"modality":"text",
									"text":"My package still has not arrived."
								}
							]
						}
					]
				},
				"tasks":[
					{
						"kind":"ConversationalSummarizationTask",
						"taskName":"summarize-conversation",
						"parameters":{"summaryAspects":["Issue","Resolution"]}
					}
				]
			}`),
		},
		{
			name:   "get analysis status",
			method: http.MethodGet,
			path:   "/azure/language/analyze-conversations/jobs/job-123?api-version=2024-11-01&showStats=true",
		},
		{
			name:   "cancel analysis job",
			method: http.MethodPost,
			path:   "/azure/language/analyze-conversations/jobs/job-123:cancel?api-version=2024-11-01",
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

func TestAzureAIServicesLanguageAnalyzeConversationsUnsupportedMethodReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/language/:analyze-conversations"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unsupported method on analyze conversations route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"azure"`) || !strings.Contains(body, path) {
		t.Fatalf("unexpected payload: %s", body)
	}
}
