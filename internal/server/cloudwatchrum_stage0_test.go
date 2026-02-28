package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func cloudWatchRUMRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "rum")
}

func cloudWatchRUMPathForTest(template string) string {
	resourceARN := "arn:aws:rum:us-east-1:123456789012:appmonitor/stackyard-app-monitor"
	out := template
	out = strings.ReplaceAll(out, "{AppMonitorName}", "stackyard-app-monitor")
	out = strings.ReplaceAll(out, "{Name}", "stackyard-app-monitor")
	out = strings.ReplaceAll(out, "{Id}", "stackyard-app-monitor")
	out = strings.ReplaceAll(out, "{ResourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestCloudWatchRUMStage0CatalogCoverage(t *testing.T) {
	if len(cloudWatchRUMOperations) != 20 {
		t.Fatalf("expected 20 CloudWatch RUM operations from docs, got %d", len(cloudWatchRUMOperations))
	}
	if len(cloudWatchRUMOperationByName) != len(cloudWatchRUMOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateAppMonitor",
		"GetAppMonitor",
		"ListAppMonitors",
		"PutRumEvents",
		"PutResourcePolicy",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := cloudWatchRUMOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(cloudWatchRUMDataTypes) != 18 {
		t.Fatalf("expected 18 CloudWatch RUM data types from docs, got %d", len(cloudWatchRUMDataTypes))
	}
	if len(cloudWatchRUMDataTypeByName) != len(cloudWatchRUMDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AppMonitor",
		"AppMonitorConfiguration",
		"MetricDefinition",
		"RumEvent",
		"TimeRange",
	}
	for _, typeName := range requiredTypes {
		if _, ok := cloudWatchRUMDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCloudWatchRUMStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchRUMRequest(t, ts, http.MethodGet, "/appmonitor/stackyard-app-monitor/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCloudWatchRUMKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchRUMRequest(t, ts, http.MethodPost, "/appmonitors", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "AppMonitorSummaries") {
		t.Fatalf("expected ListAppMonitors response body to include AppMonitorSummaries, got %q", body)
	}
}

func TestCloudWatchRUMAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cloudWatchRUMOperations {
		path := cloudWatchRUMPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := cloudWatchRUMRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
