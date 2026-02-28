package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func diagnosticToolsRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "Troubleshooting." + action,
		},
		"troubleshooting",
	)
}

func TestDiagnosticToolsStage0CatalogCoverage(t *testing.T) {
	if len(diagnosticToolsOperations) != 9 {
		t.Fatalf("expected 9 Diagnostic Tools actions from docs, got %d", len(diagnosticToolsOperations))
	}
	if len(diagnosticToolsOperationByName) != len(diagnosticToolsOperations) {
		t.Fatalf("expected unique action names")
	}

	requiredActions := []string{
		"ListTools",
		"GetTool",
		"StartExecution",
		"GetExecution",
		"GetExecutionOutput",
		"ListExecutions",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := diagnosticToolsOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(diagnosticToolsDataTypes) != 7 {
		t.Fatalf("expected 7 Diagnostic Tools data types from docs, got %d", len(diagnosticToolsDataTypes))
	}
	if len(diagnosticToolsDataTypeByName) != len(diagnosticToolsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Execution",
		"ExecutionSummary",
		"Tag",
		"Tool",
		"ToolSummary",
		"ToolVersion",
		"ValidationExceptionField",
	}
	for _, typeName := range requiredTypes {
		if _, ok := diagnosticToolsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDiagnosticToolsStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := diagnosticToolsRequest(t, ts, "UnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDiagnosticToolsStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := diagnosticToolsRequest(t, ts, "ListTools", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "tools") {
		t.Fatalf("expected ListTools response body to include tools, got %q", body)
	}
}

func TestDiagnosticToolsStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range diagnosticToolsOperations {
		resp := diagnosticToolsRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
