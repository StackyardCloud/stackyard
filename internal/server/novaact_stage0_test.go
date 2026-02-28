package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func novaActRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "nova-act")
}

func TestNovaActStage0CatalogCoverage(t *testing.T) {
	if len(novaActOperations) != 16 {
		t.Fatalf("expected 16 Nova Act operations from docs, got %d", len(novaActOperations))
	}
	if len(novaActOperationByName) != len(novaActOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateWorkflowDefinition",
		"GetWorkflowDefinition",
		"ListWorkflowDefinitions",
		"CreateWorkflowRun",
		"GetWorkflowRun",
		"ListWorkflowRuns",
		"CreateSession",
		"ListSessions",
		"CreateAct",
		"UpdateAct",
		"InvokeActStep",
		"ListActs",
		"ListModels",
	}
	for _, action := range requiredActions {
		if _, ok := novaActOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(novaActResources) != 18 {
		t.Fatalf("expected 18 Nova Act resources from docs, got %d", len(novaActResources))
	}
	if len(novaActResourceByName) != len(novaActResources) {
		t.Fatalf("expected unique resource names")
	}

	requiredResources := []string{
		"ActSummary",
		"CallResult",
		"ModelSummary",
		"SessionSummary",
		"TraceLocation",
		"WorkflowRunSummary",
	}
	for _, resourceName := range requiredResources {
		if _, ok := novaActResourceByName[resourceName]; !ok {
			t.Fatalf("missing documented resource %s", resourceName)
		}
	}
}

func TestNovaActStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := novaActRequest(t, ts, http.MethodPost, "/nova-act-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestNovaActStage0KnownActionReturnsWorkflowDefinitions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := novaActRequest(t, ts, http.MethodPost, "/workflow-definitions?maxResults=10", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "workflowDefinition") {
		t.Fatalf("expected ListWorkflowDefinitions response body to include workflowDefinition* fields, got %q", body)
	}
}

func TestNovaActStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"workflowDefinitionName":     "stackyard-workflow",
		"workflowRunId":              "workflow-run-000001",
		"sessionId":                  "session-000001",
		"actId":                      "act-000001",
		"maxResults":                 "10",
		"nextToken":                  "token-000001",
		"clientCompatibilityVersion": "1.0",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range novaActOperations {
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

		resp := novaActRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
