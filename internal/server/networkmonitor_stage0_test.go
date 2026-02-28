package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func networkMonitorRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "networkmonitor")
}

func networkMonitorPathForTest(template string) string {
	resourceARN := "arn:aws:networkmonitor:us-east-1:123456789012:monitor/stackyard-monitor"
	out := template
	out = strings.ReplaceAll(out, "{monitorName}", "stackyard-monitor")
	out = strings.ReplaceAll(out, "{probeId}", "probe-000001")
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestNetworkMonitorStage0CatalogCoverage(t *testing.T) {
	if len(networkMonitorOperations) != 12 {
		t.Fatalf("expected 12 Network Monitor operations from docs, got %d", len(networkMonitorOperations))
	}
	if len(networkMonitorOperationByName) != len(networkMonitorOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateMonitor",
		"CreateProbe",
		"GetMonitor",
		"GetProbe",
		"ListMonitors",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := networkMonitorOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(networkMonitorDataTypes) != 5 {
		t.Fatalf("expected 5 Network Monitor data types from docs, got %d", len(networkMonitorDataTypes))
	}
	if len(networkMonitorDataTypeByName) != len(networkMonitorDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"CreateMonitorProbeInput",
		"MonitorSummary",
		"Probe",
		"ProbeInput",
		"UpdateProbe",
	}
	for _, typeName := range requiredTypes {
		if _, ok := networkMonitorDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestNetworkMonitorStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := networkMonitorRequest(t, ts, http.MethodGet, "/monitors/stackyard-monitor/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestNetworkMonitorKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := networkMonitorRequest(t, ts, http.MethodGet, "/monitors", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Monitors") {
		t.Fatalf("expected ListMonitors response body to include Monitors, got %q", body)
	}
}

func TestNetworkMonitorAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range networkMonitorOperations {
		path := networkMonitorPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
			body = []byte(`{}`)
		}
		resp := networkMonitorRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
