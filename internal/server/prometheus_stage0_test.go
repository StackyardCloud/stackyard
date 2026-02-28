package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func prometheusRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "aps")
}

func prometheusPathForTest(template string) string {
	resourceARN := "arn:aws:aps:us-east-1:123456789012:workspace/ws-00000000-0000-0000-0000-000000000000"
	out := template
	out = strings.ReplaceAll(out, "{workspaceId}", "ws-00000000-0000-0000-0000-000000000000")
	out = strings.ReplaceAll(out, "{anomalyDetectorId}", "ad-000001")
	out = strings.ReplaceAll(out, "{name}", "stackyard-rules")
	out = strings.ReplaceAll(out, "{scraperId}", "scr-000001")
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestPrometheusStage0CatalogCoverage(t *testing.T) {
	if len(prometheusOperations) != 44 {
		t.Fatalf("expected 44 Prometheus operations from docs, got %d", len(prometheusOperations))
	}
	if len(prometheusOperationByName) != len(prometheusOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateWorkspace",
		"DescribeWorkspace",
		"ListWorkspaces",
		"CreateScraper",
		"GetDefaultScraperConfiguration",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := prometheusOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(prometheusDataTypes) != 42 {
		t.Fatalf("expected 42 Prometheus data types from docs, got %d", len(prometheusDataTypes))
	}
	if len(prometheusDataTypeByName) != len(prometheusDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"WorkspaceDescription",
		"WorkspaceSummary",
		"RuleGroupsNamespaceDescription",
		"ScraperDescription",
		"AnomalyDetectorDescription",
	}
	for _, typeName := range requiredTypes {
		if _, ok := prometheusDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestPrometheusStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := prometheusRequest(t, ts, http.MethodGet, "/prometheus/UnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestPrometheusKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := prometheusRequest(t, ts, http.MethodGet, "/workspaces", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "workspaces") {
		t.Fatalf("expected ListWorkspaces response body to include workspaces, got %q", body)
	}
}

func TestPrometheusAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range prometheusOperations {
		path := prometheusPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := prometheusRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
