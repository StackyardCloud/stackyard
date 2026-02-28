package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mwaaServerlessRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AmazonMWAAServerless." + action,
		},
		"mwaa-serverless",
	)
}

func TestMWAAServerlessStage0CatalogCoverage(t *testing.T) {
	if len(mwaaServerlessOperations) != 15 {
		t.Fatalf("expected 15 MWAA Serverless operations from docs, got %d", len(mwaaServerlessOperations))
	}
	if len(mwaaServerlessOperationByName) != len(mwaaServerlessOperations) {
		t.Fatalf("expected unique MWAA Serverless operation names")
	}

	requiredActions := []string{
		"CreateWorkflow",
		"DeleteWorkflow",
		"GetTaskInstance",
		"GetWorkflow",
		"GetWorkflowRun",
		"ListTaskInstances",
		"ListWorkflowRuns",
		"ListWorkflowVersions",
		"ListWorkflows",
		"StartWorkflowRun",
		"StopWorkflowRun",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
		"UpdateWorkflow",
	}
	for _, action := range requiredActions {
		if _, ok := mwaaServerlessOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(mwaaServerlessDataTypes) != 12 {
		t.Fatalf("expected 12 MWAA Serverless data types from docs, got %d", len(mwaaServerlessDataTypes))
	}
	if len(mwaaServerlessDataTypeByName) != len(mwaaServerlessDataTypes) {
		t.Fatalf("expected unique MWAA Serverless data type names")
	}

	requiredTypes := []string{
		"DefinitionS3Location",
		"EncryptionConfiguration",
		"LoggingConfiguration",
		"NetworkConfiguration",
		"WorkflowRunDetail",
		"WorkflowRunSummary",
		"WorkflowSummary",
		"WorkflowVersionSummary",
	}
	for _, typeName := range requiredTypes {
		if _, ok := mwaaServerlessDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestMWAAServerlessStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mwaaServerlessRequest(t, ts, "TotallyUnknownMWAAAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMWAAServerlessStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mwaaServerlessRequest(t, ts, "ListWorkflows", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Workflows") {
		t.Fatalf("expected ListWorkflows response body to include Workflows, got %q", body)
	}
}

func TestMWAAServerlessStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range mwaaServerlessOperations {
		resp := mwaaServerlessRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
