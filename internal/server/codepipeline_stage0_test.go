package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func codePipelineRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "CodePipeline_20150709." + action,
		},
		"codepipeline",
	)
}

func TestCodePipelineStage0CatalogCoverage(t *testing.T) {
	if len(codePipelineOperations) != 44 {
		t.Fatalf("expected 44 CodePipeline operations from docs, got %d", len(codePipelineOperations))
	}
	if len(codePipelineOperationByName) != len(codePipelineOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreatePipeline",
		"GetPipeline",
		"ListPipelines",
		"StartPipelineExecution",
		"StopPipelineExecution",
		"ListDeployActionExecutionTargets",
	}
	for _, action := range requiredActions {
		if _, ok := codePipelineOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(codePipelineDataTypes) != 112 {
		t.Fatalf("expected 112 CodePipeline data types from docs, got %d", len(codePipelineDataTypes))
	}
	if len(codePipelineDataTypeByName) != len(codePipelineDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"PipelineDeclaration",
		"PipelineExecutionSummary",
		"StageDeclaration",
		"ActionDeclaration",
		"WebhookDefinition",
		"DeployActionExecutionTarget",
	}
	for _, typeName := range requiredTypes {
		if _, ok := codePipelineDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCodePipelineStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codePipelineRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCodePipelineStage0KnownActionDoesNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codePipelineRequest(t, ts, "ListPipelines", `{"MaxResults":10}`)
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected ListPipelines to be implemented")
	}
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected non-NotImplemented response body, got %q", body)
	}
}

func TestCodePipelineStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range codePipelineOperations {
		resp := codePipelineRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
