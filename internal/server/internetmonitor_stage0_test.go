package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func internetMonitorRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "internetmonitor")
}

func internetMonitorPathForTest(template string) string {
	resourceARN := "arn:aws:internetmonitor:us-east-1:123456789012:monitor/stackyard-monitor"
	out := template
	out = strings.ReplaceAll(out, "{MonitorName}", "stackyard-monitor")
	out = strings.ReplaceAll(out, "{EventId}", "event-000001")
	out = strings.ReplaceAll(out, "{QueryId}", "query-000001")
	out = strings.ReplaceAll(out, "{ResourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestInternetMonitorStage0CatalogCoverage(t *testing.T) {
	if len(internetMonitorOperations) != 16 {
		t.Fatalf("expected 16 Internet Monitor operations from docs, got %d", len(internetMonitorOperations))
	}
	if len(internetMonitorOperationByName) != len(internetMonitorOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateMonitor",
		"GetMonitor",
		"ListMonitors",
		"StartQuery",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := internetMonitorOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(internetMonitorDataTypes) != 18 {
		t.Fatalf("expected 18 Internet Monitor data types from docs, got %d", len(internetMonitorDataTypes))
	}
	if len(internetMonitorDataTypeByName) != len(internetMonitorDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Monitor",
		"HealthEvent",
		"InternetEventSummary",
		"NetworkImpairment",
	}
	for _, typeName := range requiredTypes {
		if _, ok := internetMonitorDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestInternetMonitorStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := internetMonitorRequest(t, ts, http.MethodGet, "/v20210603/Monitors/stackyard-monitor/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestInternetMonitorKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := internetMonitorRequest(t, ts, http.MethodGet, "/v20210603/Monitors", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Monitors") {
		t.Fatalf("expected ListMonitors response body to include Monitors, got %q", body)
	}
}

func TestInternetMonitorAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range internetMonitorOperations {
		path := internetMonitorPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
			body = []byte(`{}`)
		}
		resp := internetMonitorRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
