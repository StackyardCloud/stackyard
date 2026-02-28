package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func bedrockAgentCoreControlRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "bedrock-agentcore")
}

func TestBedrockAgentCoreControlStage0CatalogCoverage(t *testing.T) {
	if len(bedrockAgentCoreControlOperations) != 86 {
		t.Fatalf("expected 86 Bedrock AgentCore Control Plane operations from docs, got %d", len(bedrockAgentCoreControlOperations))
	}
	if len(bedrockAgentCoreControlOperationByName) != len(bedrockAgentCoreControlOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateAgentRuntime",
		"GetAgentRuntime",
		"ListAgentRuntimes",
		"CreatePolicyEngine",
		"StartPolicyGeneration",
		"PutResourcePolicy",
		"TagResource",
		"UpdateWorkloadIdentity",
	}
	for _, action := range requiredActions {
		if _, ok := bedrockAgentCoreControlOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(bedrockAgentCoreControlTypes) != 175 {
		t.Fatalf("expected 175 Bedrock AgentCore Control Plane data types from docs, got %d", len(bedrockAgentCoreControlTypes))
	}
	if len(bedrockAgentCoreControlTypeByName) != len(bedrockAgentCoreControlTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AgentRuntime",
		"GatewaySummary",
		"Memory",
		"PolicyEngine",
		"PolicyGeneration",
		"WorkloadIdentityDetails",
	}
	for _, typeName := range requiredTypes {
		if _, ok := bedrockAgentCoreControlTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestBedrockAgentCoreControlStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/gateways/gateway-000001/unsupported", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestBedrockAgentCoreControlStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/runtimes/?maxResults=10&nextToken=", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "agentRuntimes") {
		t.Fatalf("expected ListAgentRuntimes response body to include agentRuntimes, got %q", body)
	}
}

func TestBedrockAgentCoreControlStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"agentRuntimeId":           "runtime-000001",
		"agentRuntimeVersion":      "v1",
		"browserId":                "browser-000001",
		"clientToken":              "stackyard-client-token-000001",
		"codeInterpreterId":        "code-000001",
		"endpointName":             "runtime-endpoint-000001",
		"evaluatorId":              "evaluator-000001",
		"gatewayIdentifier":        "gateway-000001",
		"maxResults":               "10",
		"memoryId":                 "memory-000001",
		"nextToken":                "",
		"onlineEvaluationConfigId": "online-eval-000001",
		"policyEngineId":           "policy-engine-000001",
		"policyGenerationId":       "policy-generation-000001",
		"policyId":                 "policy-000001",
		"profileId":                "profile-000001",
		"resourceArn":              "arn:aws:bedrock-agentcore:us-east-1:123456789012:resource/stackyard-control",
		"tagKeys":                  "env",
		"targetId":                 "target-000001",
		"targetResourceScope":      "ALL",
		"type":                     "SYSTEM",
		"view":                     "DETAIL",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range bedrockAgentCoreControlOperations {
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

		resp := bedrockAgentCoreControlRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
