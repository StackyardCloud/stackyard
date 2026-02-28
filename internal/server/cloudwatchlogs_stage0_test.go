package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cloudWatchLogsRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "logs")
}

func TestCloudWatchLogsStage0CatalogCoverage(t *testing.T) {
	if len(cloudWatchLogsOperations) != 107 {
		t.Fatalf("expected 107 CloudWatch Logs operations from docs, got %d", len(cloudWatchLogsOperations))
	}
	if len(cloudWatchLogsOperationByName) != len(cloudWatchLogsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateLogGroup",
		"CreateLogStream",
		"PutLogEvents",
		"GetLogEvents",
		"FilterLogEvents",
		"StartQuery",
		"DescribeLogGroups",
	}
	for _, action := range requiredActions {
		if _, ok := cloudWatchLogsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(cloudWatchLogsDataTypes) != 113 {
		t.Fatalf("expected 113 CloudWatch Logs data types from docs, got %d", len(cloudWatchLogsDataTypes))
	}
	if len(cloudWatchLogsDataTypeByName) != len(cloudWatchLogsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"LogGroup",
		"LogStream",
		"InputLogEvent",
		"OutputLogEvent",
		"QueryInfo",
	}
	for _, typeName := range requiredTypes {
		if _, ok := cloudWatchLogsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCloudWatchLogsStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchLogsRequest(t, ts, http.MethodPost, "/UnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCloudWatchLogsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchLogsRequest(t, ts, http.MethodPost, "/DescribeLogGroups", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "logGroups") {
		t.Fatalf("expected DescribeLogGroups response body to include logGroups, got %q", body)
	}
}

func TestCloudWatchLogsAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cloudWatchLogsOperations {
		body := []byte(`{}`)
		resp := cloudWatchLogsRequest(t, ts, op.Method, op.URI, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
