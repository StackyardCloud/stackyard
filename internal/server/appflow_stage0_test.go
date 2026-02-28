package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func appFlowRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "appflow")
}

func TestAppFlowStage0CatalogCoverage(t *testing.T) {
	if len(appFlowOperations) != 25 {
		t.Fatalf("expected 25 AppFlow operations from docs, got %d", len(appFlowOperations))
	}
	if len(appFlowOperationByName) != len(appFlowOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateFlow",
		"UpdateFlow",
		"DeleteFlow",
		"DescribeFlow",
		"ListFlows",
		"StartFlow",
		"StopFlow",
		"ListTagsForResource",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := appFlowOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(appFlowDataTypes) != 151 {
		t.Fatalf("expected 151 AppFlow data types from docs, got %d", len(appFlowDataTypes))
	}
	if len(appFlowDataTypeByName) != len(appFlowDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"FlowDefinition",
		"ConnectorProfile",
		"TriggerConfig",
		"Task",
		"DestinationFlowConfig",
		"SourceFlowConfig",
	}
	for _, typeName := range requiredTypes {
		if _, ok := appFlowDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestAppFlowUnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := appFlowRequest(t, ts, http.MethodPost, "/appflow-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAppFlowKnownActionReturnsFlowList(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := appFlowRequest(t, ts, http.MethodPost, "/list-flows", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "flows") {
		t.Fatalf("expected ListFlows response to include flows, got %q", body)
	}
}

func TestAppFlowAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := url.PathEscape("arn:aws:appflow:us-east-1:123456789012:flow/stackyard-seed-flow")
	for _, op := range appFlowOperations {
		path := strings.ReplaceAll(op.URI, "{resourceArn}", resourceARN)
		path = strings.ReplaceAll(path, "{tagKeys}", "env")

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := appFlowRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
