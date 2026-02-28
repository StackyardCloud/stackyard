package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiagnosticToolsStage12ToolDiscoveryAndExecutionLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	cases := []struct {
		name    string
		action  string
		payload string
	}{
		{name: "ListTools", action: "ListTools", payload: `{}`},
		{name: "GetTool", action: "GetTool", payload: `{"toolId":"EC2SystemsManager"}`},
		{name: "StartExecution", action: "StartExecution", payload: `{"toolId":"EC2SystemsManager","toolVersionId":"1.0.0","targetRegions":["us-east-1"],"storageRegion":"us-east-1"}`},
		{name: "GetExecution", action: "GetExecution", payload: `{"executionId":"e-000002"}`},
		{name: "GetExecutionOutput", action: "GetExecutionOutput", payload: `{"executionId":"e-000002"}`},
		{name: "ListExecutions", action: "ListExecutions", payload: `{"maxResults":10}`},
	}

	for _, tc := range cases {
		resp := diagnosticToolsRequest(t, ts, tc.action, tc.payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tc.name, resp.StatusCode, body)
		}
		if strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s: expected non-NotImplemented response, got: %s", tc.name, body)
		}
	}
}

func TestDiagnosticToolsStage345TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := diagnosticToolsRequest(t, ts, "TagResource", `{"identifier":"e-000001","tags":[{"key":"env","value":"stage"},{"key":"owner","value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = diagnosticToolsRequest(t, ts, "ListTagsForResource", `{"identifier":"e-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = diagnosticToolsRequest(t, ts, "UntagResource", `{"identifier":"e-000001","tagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = diagnosticToolsRequest(t, ts, "ListTagsForResource", `{"identifier":"e-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if strings.Contains(body, "\"owner\"") {
		t.Fatalf("expected owner tag to be removed, got %q", body)
	}

	// Idempotency: repeated StartExecution and TagResource calls should remain successful.
	resp = diagnosticToolsRequest(t, ts, "StartExecution", `{"executionId":"e-000777","toolId":"EC2SystemsManager"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = diagnosticToolsRequest(t, ts, "StartExecution", `{"executionId":"e-000777","toolId":"EC2SystemsManager"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = diagnosticToolsRequest(t, ts, "TagResource", `{"identifier":"e-000777","tags":[{"key":"env","value":"stage"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = diagnosticToolsRequest(t, ts, "TagResource", `{"identifier":"e-000777","tags":[{"key":"env","value":"stage"}]}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestDiagnosticToolsStage6MalformedBodyReturnsValidation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "Troubleshooting.ListTools",
		},
		"troubleshooting",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
