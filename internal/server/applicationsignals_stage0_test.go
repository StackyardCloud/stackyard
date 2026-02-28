package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func applicationSignalsRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "application-signals")
}

func applicationSignalsPathForTest(template string) string {
	out := template
	out = strings.ReplaceAll(out, "{Id}", "stackyard-slo")
	return out
}

func TestApplicationSignalsStage0CatalogCoverage(t *testing.T) {
	if len(applicationSignalsOperations) != 23 {
		t.Fatalf("expected 23 Application Signals operations from docs, got %d", len(applicationSignalsOperations))
	}
	if len(applicationSignalsOperationByName) != len(applicationSignalsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateServiceLevelObjective",
		"GetServiceLevelObjective",
		"ListServices",
		"ListAuditFindings",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := applicationSignalsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(applicationSignalsDataTypes) != 52 {
		t.Fatalf("expected 52 Application Signals data types from docs, got %d", len(applicationSignalsDataTypes))
	}
	if len(applicationSignalsDataTypeByName) != len(applicationSignalsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ServiceLevelObjective",
		"Service",
		"ServiceDependency",
		"AuditFinding",
		"GroupingConfiguration",
	}
	for _, typeName := range requiredTypes {
		if _, ok := applicationSignalsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestApplicationSignalsStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := applicationSignalsRequest(t, ts, http.MethodGet, "/slo/stackyard-slo/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestApplicationSignalsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := applicationSignalsRequest(t, ts, http.MethodGet, "/services", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "ServiceSummaries") {
		t.Fatalf("expected ListServices response body to include ServiceSummaries, got %q", body)
	}
}

func TestApplicationSignalsAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range applicationSignalsOperations {
		path := applicationSignalsPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
			body = []byte(`{}`)
		}
		resp := applicationSignalsRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
