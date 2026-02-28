package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func migrationHubOrchestratorRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "migrationhub-orchestrator")
}

func migrationHubOrchestratorPathForOperation(op migrationHubOrchestratorOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{id}":          "mwf-00000001",
		"{templateId}":  "tmpl-00000001",
		"{workflowId}":  "mwf-00000001",
		"{stepGroupId}": "wsg-00000001",
		"{resourceArn}": url.PathEscape("arn:aws:migrationhub-orchestrator:us-east-1:123456789012:workflow/mwf-00000001"),
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	if op.Name == "GetTemplate" || op.Name == "DeleteTemplate" || op.Name == "UpdateTemplate" || op.Name == "GetTemplateStep" || op.Name == "DeleteWorkflowStep" || op.Name == "UpdateWorkflowStep" || op.Name == "RetryWorkflowStep" {
		path = strings.ReplaceAll(path, "mwf-00000001", "tmpl-00000001")
	}
	if op.Name == "GetWorkflowStepGroup" || op.Name == "DeleteWorkflowStepGroup" || op.Name == "UpdateWorkflowStepGroup" {
		path = strings.ReplaceAll(path, "mwf-00000001", "wsg-00000001")
	}
	if op.Name == "GetWorkflowStep" || op.Name == "DeleteWorkflowStep" || op.Name == "UpdateWorkflowStep" || op.Name == "RetryWorkflowStep" {
		path = strings.ReplaceAll(path, "tmpl-00000001", "wstep-00000001")
	}
	return path
}

func TestMigrationHubOrchestratorStage0CatalogCoverage(t *testing.T) {
	if len(migrationHubOrchestratorOperations) != 31 {
		t.Fatalf("expected 31 Migration Hub Orchestrator operations from docs, got %d", len(migrationHubOrchestratorOperations))
	}
	if len(migrationHubOrchestratorOperationByName) != len(migrationHubOrchestratorOperations) {
		t.Fatalf("expected unique Migration Hub Orchestrator operation names")
	}

	requiredActions := []string{
		"CreateTemplate",
		"CreateWorkflow",
		"CreateWorkflowStep",
		"GetTemplate",
		"GetWorkflow",
		"ListPlugins",
		"ListWorkflows",
		"RetryWorkflowStep",
		"StartWorkflow",
		"StopWorkflow",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := migrationHubOrchestratorOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(migrationHubOrchestratorDataTypes) != 18 {
		t.Fatalf("expected 18 Migration Hub Orchestrator data types from docs, got %d", len(migrationHubOrchestratorDataTypes))
	}
	if len(migrationHubOrchestratorDataTypeByName) != len(migrationHubOrchestratorDataTypes) {
		t.Fatalf("expected unique Migration Hub Orchestrator data type names")
	}
}

func TestMigrationHubOrchestratorStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := migrationHubOrchestratorRequest(t, ts, http.MethodGet, "/not-a-real-orchestrator-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMigrationHubOrchestratorStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := migrationHubOrchestratorRequest(t, ts, http.MethodGet, "/migrationworkflowtemplates", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "templateSummary") {
		t.Fatalf("expected ListTemplates response body to include templateSummary, got %q", body)
	}
}

func TestMigrationHubOrchestratorStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range migrationHubOrchestratorOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := migrationHubOrchestratorRequest(t, ts, op.Method, migrationHubOrchestratorPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
