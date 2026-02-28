package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func cloudWatchSyntheticsRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "synthetics")
}

func cloudWatchSyntheticsPathForTest(template string) string {
	out := template
	out = strings.ReplaceAll(out, "{name}", "stackyard-canary")
	out = strings.ReplaceAll(out, "{groupIdentifier}", "stackyard-group")
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape("arn:aws:synthetics:us-east-1:123456789012:canary:stackyard-canary"))
	return out
}

func TestCloudWatchSyntheticsStage0CatalogCoverage(t *testing.T) {
	if len(cloudWatchSyntheticsOperations) != 22 {
		t.Fatalf("expected 22 CloudWatch Synthetics operations from docs, got %d", len(cloudWatchSyntheticsOperations))
	}
	if len(cloudWatchSyntheticsOperationByName) != len(cloudWatchSyntheticsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateCanary",
		"UpdateCanary",
		"DescribeCanaries",
		"CreateGroup",
		"DescribeRuntimeVersions",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := cloudWatchSyntheticsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(cloudWatchSyntheticsDataTypes) != 32 {
		t.Fatalf("expected 32 CloudWatch Synthetics data types from docs, got %d", len(cloudWatchSyntheticsDataTypes))
	}
	if len(cloudWatchSyntheticsDataTypeByName) != len(cloudWatchSyntheticsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Canary",
		"CanaryRun",
		"Group",
		"RuntimeVersion",
		"ArtifactConfigInput",
	}
	for _, typeName := range requiredTypes {
		if _, ok := cloudWatchSyntheticsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCloudWatchSyntheticsStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchSyntheticsRequest(t, ts, http.MethodGet, "/canary/stackyard-canary/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCloudWatchSyntheticsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchSyntheticsRequest(t, ts, http.MethodPost, "/canaries", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Canaries") {
		t.Fatalf("expected DescribeCanaries response body to include Canaries, got %q", body)
	}
}

func TestCloudWatchSyntheticsAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cloudWatchSyntheticsOperations {
		path := cloudWatchSyntheticsPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := cloudWatchSyntheticsRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
