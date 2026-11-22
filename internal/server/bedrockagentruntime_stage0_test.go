package server

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

var barStreamingOps = map[string]bool{
	"InvokeAgent": true, "InvokeInlineAgent": true, "InvokeFlow": true,
	"RetrieveAndGenerateStream": true, "OptimizePrompt": true, "AgenticRetrieveStream": true,
}

func TestBedrockAgentRuntimeStage0CatalogCoverage(t *testing.T) {
	if len(bedrockagentruntimeOperations) != 35 {
		t.Fatalf("expected 35 bedrock-agent-runtime operations, got %d", len(bedrockagentruntimeOperations))
	}
	if len(bedrockagentruntimeOperationByName) != len(bedrockagentruntimeOperations) {
		t.Fatal("expected unique operation names")
	}
	for _, op := range bedrockagentruntimeOperations {
		if barHandlers[op.Name] == nil && !barStreamingOps[op.Name] {
			t.Errorf("%s has no handler", op.Name)
		}
	}
	if len(barHandlers)+len(barStreamingOps) != len(bedrockagentruntimeOperations) {
		t.Fatalf("handlers (%d+%d) and operations (%d) differ", len(barHandlers), len(barStreamingOps), len(bedrockagentruntimeOperations))
	}
}

func TestBedrockAgentRuntimeStage0RoutesAreUnambiguous(t *testing.T) {
	param := regexp.MustCompile(`\{[^}]+\}`)
	for _, op := range bedrockagentruntimeOperations {
		path := param.ReplaceAllString(op.URI, "ABCDEFGHIJ")
		got, params := matchBedrockAgentRuntimeRoute(op.Method, path)
		if got == nil || got.Name != op.Name {
			t.Errorf("%s %s routed to %+v", op.Method, path, got)
			continue
		}
		if strings.Count(op.URI, "{") != len(params) {
			t.Errorf("%s: expected %d params, got %v", op.Name, strings.Count(op.URI, "{"), params)
		}
	}
}

// The control plane, Bedrock Runtime and AgentCore keep their traffic, and
// PrepareAgent (POST /agents/{id}/) is not mistaken for InvokeInlineAgent
// (POST /agents/{sessionId}).
func TestBedrockAgentRuntimeStage0OtherBedrockRoutersKeepTheirTraffic(t *testing.T) {
	ts := newBedrockAgentTestServer(t)

	id := createPreparedAgent(t, ts, "router", "DISABLED")
	if s := baObj(t, baWant(t, ts, http.StatusOK, http.MethodGet, "/agents/"+id+"/", ""), "agent")["agentStatus"]; s != "PREPARED" {
		t.Fatalf("PrepareAgent must still reach bedrock-agent, agent is %v", s)
	}
	status, _, raw := brtCall(t, ts, http.MethodPost, "/agents/inline-session-1", map[string]any{
		"foundationModel": "anthropic.claude-v2", "instruction": bedrockAgentTestInstruction, "inputText": "hello there",
	}, nil)
	if status != http.StatusOK {
		t.Fatalf("InvokeInlineAgent: %d %s", status, raw)
	}
	if types, _ := brtFrames(t, raw); len(types) != 1 || types[0] != "chunk" {
		t.Fatalf("InvokeInlineAgent should stream one chunk, got %v", types)
	}

	// Bedrock Runtime (also signed "bedrock") still answers model calls.
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("anthropic.claude-3-haiku-20240307-v1:0", "invoke"), map[string]any{
		"anthropic_version": "bedrock-2023-05-31", "max_tokens": 10, "messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}, nil)
	if status != http.StatusOK {
		t.Fatalf("bedrock runtime InvokeModel: %d %s", status, raw)
	}

	// Bedrock control plane.
	resp := bedrockRequest(t, ts, "ListFoundationModels", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "modelSummaries") {
		t.Fatalf("bedrock control plane response changed: %s", body)
	}

	// AgentCore control plane.
	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/runtimes/", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, "sessionSummaries") {
		t.Fatalf("AgentCore traffic reached bedrock-agent-runtime: %s", body)
	}

	// A bucket called "sessions" is still S3.
	resp = signedRequest(t, http.MethodPut, ts.URL+"/sessions", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)
}

func TestBedrockAgentRuntimeStage0RejectsWrongSigningService(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	req := httptest.NewRequest(http.MethodPost, "/sessions/", strings.NewReader(`{}`))
	req.Host = "bedrock-agent-runtime.us-east-1.amazonaws.com"
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=stackyard/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=x")
	req.Header.Set("X-Amz-Date", "20240101T000000Z")
	rec := httptest.NewRecorder()
	if !srv.handleBedrockAgentRuntimeRESTRouter(rec, req) {
		t.Fatal("a bedrock-agent-runtime host must be claimed")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a request signed for s3, got %d: %s", rec.Code, rec.Body.String())
	}
	// The bedrock-agent router leaves bedrock-agent-runtime hosts alone.
	req = httptest.NewRequest(http.MethodPost, "/agents/ABCDEFGHIJ/", nil)
	req.Host = "bedrock-agent-runtime.us-east-1.amazonaws.com"
	if srv.handleBedrockAgentRESTRouter(httptest.NewRecorder(), req) {
		t.Fatal("bedrock-agent must not claim a bedrock-agent-runtime host")
	}
}
