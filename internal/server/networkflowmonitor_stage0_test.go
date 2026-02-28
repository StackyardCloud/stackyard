package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func networkFlowMonitorRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "networkflowmonitor")
}

func networkFlowMonitorPathForTest(template string) string {
	resourceARN := "arn:aws:networkflowmonitor:us-east-1:123456789012:monitor/stackyard-monitor"
	out := template
	out = strings.ReplaceAll(out, "{monitorName}", "stackyard-monitor")
	out = strings.ReplaceAll(out, "{scopeId}", "scope-00000000000000001")
	out = strings.ReplaceAll(out, "{queryId}", "query-000001")
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestNetworkFlowMonitorStage0CatalogCoverage(t *testing.T) {
	if len(networkFlowMonitorOperations) != 25 {
		t.Fatalf("expected 25 Network Flow Monitor operations from docs, got %d", len(networkFlowMonitorOperations))
	}
	if len(networkFlowMonitorOperationByName) != len(networkFlowMonitorOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateMonitor",
		"CreateScope",
		"GetMonitor",
		"ListMonitors",
		"StartQueryMonitorTopContributors",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := networkFlowMonitorOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(networkFlowMonitorDataTypes) != 12 {
		t.Fatalf("expected 12 Network Flow Monitor data types from docs, got %d", len(networkFlowMonitorDataTypes))
	}
	if len(networkFlowMonitorDataTypeByName) != len(networkFlowMonitorDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"MonitorSummary",
		"ScopeSummary",
		"TargetIdentifier",
		"TraversedComponent",
	}
	for _, typeName := range requiredTypes {
		if _, ok := networkFlowMonitorDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestNetworkFlowMonitorStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := networkFlowMonitorRequest(t, ts, http.MethodGet, "/monitors/stackyard-monitor/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestNetworkFlowMonitorKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := networkFlowMonitorRequest(t, ts, http.MethodGet, "/monitors", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Monitors") {
		t.Fatalf("expected ListMonitors response body to include Monitors, got %q", body)
	}
}

func TestNetworkFlowMonitorAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range networkFlowMonitorOperations {
		path := networkFlowMonitorPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
			body = []byte(`{}`)
		}
		resp := networkFlowMonitorRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
