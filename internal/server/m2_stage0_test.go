package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func m2Request(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "m2")
}

func m2PathForOperation(op m2Operation) string {
	path := op.URI
	replacements := map[string]string{
		"{applicationId}":      "app-00000001",
		"{environmentId}":      "env-00000001",
		"{deploymentId}":       "dep-00000001",
		"{applicationVersion}": "1",
		"{executionId}":        "exec-00000001",
		"{taskId}":             "task-00000001",
		"{dataSetName}":        "SYS1.PARMLIB",
		"{resourceArn}":        url.PathEscape("arn:aws:m2:us-east-1:123456789012:application/app-00000001"),
	}
	for k, v := range replacements {
		path = strings.ReplaceAll(path, k, v)
	}
	return path
}

func TestM2Stage0CatalogCoverage(t *testing.T) {
	if len(m2Operations) != 37 {
		t.Fatalf("expected 37 M2 operations from docs, got %d", len(m2Operations))
	}
	if len(m2OperationByName) != len(m2Operations) {
		t.Fatalf("expected unique M2 operation names")
	}

	requiredActions := []string{
		"CreateApplication",
		"GetApplication",
		"ListApplications",
		"CreateEnvironment",
		"GetEnvironment",
		"ListEnvironments",
		"CreateDeployment",
		"ListDeployments",
		"StartBatchJob",
		"CancelBatchJobExecution",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := m2OperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(m2DataTypes) != 52 {
		t.Fatalf("expected 52 M2 data types from docs, got %d", len(m2DataTypes))
	}
	if len(m2DataTypeByName) != len(m2DataTypes) {
		t.Fatalf("expected unique M2 data type names")
	}
}

func TestM2Stage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := m2Request(t, ts, http.MethodPost, "/unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestM2Stage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := m2Request(t, ts, http.MethodGet, "/applications", ``)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "applications") {
		t.Fatalf("expected ListApplications response body to include applications, got %q", body)
	}
}

func TestM2Stage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range m2Operations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := m2Request(t, ts, op.Method, m2PathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
