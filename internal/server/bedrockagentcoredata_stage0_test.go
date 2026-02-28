package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func bedrockAgentCoreDataRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "bedrock-agentcore")
}

func TestBedrockAgentCoreDataStage0CatalogCoverage(t *testing.T) {
	if len(bedrockAgentCoreDataOperations) != 36 {
		t.Fatalf("expected 36 Bedrock AgentCore Data Plane operations from docs, got %d", len(bedrockAgentCoreDataOperations))
	}
	if len(bedrockAgentCoreDataOperationByName) != len(bedrockAgentCoreDataOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"BatchCreateMemoryRecords",
		"GetWorkloadAccessToken",
		"InvokeAgentRuntime",
		"ListMemoryRecords",
		"RetrieveMemoryRecords",
		"StartBrowserSession",
		"StopRuntimeSession",
		"UpdateBrowserStream",
	}
	for _, action := range requiredActions {
		if _, ok := bedrockAgentCoreDataOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(bedrockAgentCoreDataTypes) != 55 {
		t.Fatalf("expected 55 Bedrock AgentCore Data Plane data types from docs, got %d", len(bedrockAgentCoreDataTypes))
	}
	if len(bedrockAgentCoreDataTypeByName) != len(bedrockAgentCoreDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ActorSummary",
		"AutomationStream",
		"MemoryRecord",
		"SessionSummary",
		"UpdateBrowserStream",
		"TokenUsage",
	}
	for _, typeName := range requiredTypes {
		if _, ok := bedrockAgentCoreDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestBedrockAgentCoreDataStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/bedrock-agentcore-unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestBedrockAgentCoreDataStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/memoryRecords", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "memoryRecords") {
		t.Fatalf("expected ListMemoryRecords response body to include memoryRecords, got %q", body)
	}
}

func TestBedrockAgentCoreDataStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"memoryId":                  "memory-000001",
		"memoryRecordId":            "record-000001",
		"actorId":                   "actor-000001",
		"sessionId":                 "session-000001",
		"eventId":                   "event-000001",
		"evaluatorId":               "evaluator-000001",
		"agentRuntimeArn":           "arn:aws:bedrock-agentcore:us-east-1:123456789012:runtime/stackyard-runtime",
		"qualifier":                 "LATEST",
		"accountId":                 "123456789012",
		"browserIdentifier":         "browser-000001",
		"codeInterpreterIdentifier": "code-000001",
		"profileIdentifier":         "profile-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range bedrockAgentCoreDataOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.QueryEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := bedrockAgentCoreDataRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
