package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func newBedrockAgentTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// bedrockAgentCall sends a bedrock-signed REST call and decodes the JSON reply.
func bedrockAgentCall(t *testing.T, ts *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	headers := map[string]string{}
	var raw []byte
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "bedrock")
	data := mustBody(t, resp)
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, path, data, err)
		}
	}
	return resp.StatusCode, out
}

func TestBedrockAgentStage0CatalogCoverage(t *testing.T) {
	if len(bedrockagentOperations) != 86 {
		t.Fatalf("expected 86 bedrock-agent operations, got %d", len(bedrockagentOperations))
	}
	if len(bedrockagentOperationByName) != len(bedrockagentOperations) {
		t.Fatalf("expected unique operation names")
	}
	for _, op := range bedrockagentOperations {
		if bedrockAgentHandlers[op.Name] == nil {
			t.Errorf("%s has no handler", op.Name)
		}
		if op.Status < 200 || op.Status > 299 {
			t.Errorf("%s has status %d", op.Name, op.Status)
		}
	}
	if len(bedrockAgentHandlers) != len(bedrockagentOperations) {
		t.Fatalf("handlers (%d) and operations (%d) differ", len(bedrockAgentHandlers), len(bedrockagentOperations))
	}
}

// Every URI template, filled with sample values, must route back to its own
// operation, so no two operations shadow each other.
func TestBedrockAgentStage0RoutesAreUnambiguous(t *testing.T) {
	param := regexp.MustCompile(`\{[^}]+\}`)
	for _, op := range bedrockagentOperations {
		path := param.ReplaceAllString(op.URI, "ABCDEFGHIJ")
		got, params := matchBedrockAgentRoute(op.Method, path)
		if got == nil || got.Name != op.Name {
			name := "<none>"
			if got != nil {
				name = got.Name
			}
			t.Errorf("%s %s routed to %s", op.Method, path, name)
			continue
		}
		if strings.Count(op.URI, "{") != len(params) {
			t.Errorf("%s: expected %d params, got %v", op.Name, strings.Count(op.URI, "{"), params)
		}
	}
	if op, _ := matchBedrockAgentRoute(http.MethodPost, "/flows/validate-definition"); op == nil || op.Name != "ValidateFlowDefinition" {
		t.Fatalf("validate-definition must not route to PrepareFlow: %+v", op)
	}
}

func TestBedrockAgentStage0OtherBedrockRoutersKeepTheirTraffic(t *testing.T) {
	ts := newBedrockAgentTestServer(t)

	// Bedrock control plane (X-Amz-Target, also signed "bedrock").
	resp := bedrockRequest(t, ts, "ListFoundationModels", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "modelSummaries") {
		t.Fatalf("bedrock control plane response changed: %s", body)
	}

	// AgentCore control signs as bedrock-agentcore and shares /tags.
	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/runtimes/", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, "agentSummaries") {
		t.Fatalf("AgentCore traffic reached bedrock-agent: %s", body)
	}

	// An S3 bucket named "agents" is still S3.
	resp = signedRequest(t, http.MethodPut, ts.URL+"/agents", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)

	// Bedrock Agent Runtime routes under the same prefixes are not claimed.
	for _, c := range []struct{ method, path string }{
		{http.MethodPost, "/agents/ABCDEFGHIJ/agentAliases/TSTALIASID/sessions/s1/text"},
		{http.MethodPost, "/knowledgebases/ABCDEFGHIJ/retrieve"},
		{http.MethodPost, "/tags/" + url.PathEscape("arn:aws:bedrock:us-east-1:123456789012:session/abc")},
	} {
		req := httptest.NewRequest(c.method, c.path, strings.NewReader(`{}`))
		req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=stackyard/20240101/us-east-1/bedrock/aws4_request, SignedHeaders=host, Signature=x")
		srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
		if srv.handleBedrockAgentRESTRouter(httptest.NewRecorder(), req) {
			t.Errorf("%s %s should be left for Bedrock Agent Runtime", c.method, c.path)
		}
	}
}

func TestBedrockAgentStage0RejectsWrongSigningService(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	req := httptest.NewRequest(http.MethodPost, "/agents/", strings.NewReader(`{}`))
	req.Host = "bedrock-agent.us-east-1.amazonaws.com"
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=stackyard/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=x")
	req.Header.Set("X-Amz-Date", "20240101T000000Z")
	rec := httptest.NewRecorder()
	if !srv.handleBedrockAgentRESTRouter(rec, req) {
		t.Fatal("a bedrock-agent host must be claimed")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a request signed for s3, got %d: %s", rec.Code, rec.Body.String())
	}
}
