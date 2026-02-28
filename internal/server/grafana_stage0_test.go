package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func grafanaRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "grafana")
}

func grafanaPathForTest(template string) string {
	resourceARN := "arn:aws:grafana:us-east-1:123456789012:/workspaces/g-0000000000"
	out := template
	out = strings.ReplaceAll(out, "{workspaceId}", "g-0000000000")
	out = strings.ReplaceAll(out, "{licenseType}", "ENTERPRISE")
	out = strings.ReplaceAll(out, "{serviceAccountId}", "sa-000001")
	out = strings.ReplaceAll(out, "{tokenId}", "sat-000001")
	out = strings.ReplaceAll(out, "{keyName}", "stackyard-key")
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestGrafanaStage0CatalogCoverage(t *testing.T) {
	if len(grafanaOperations) != 25 {
		t.Fatalf("expected 25 Grafana operations from docs, got %d", len(grafanaOperations))
	}
	if len(grafanaOperationByName) != len(grafanaOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateWorkspace",
		"DescribeWorkspace",
		"ListWorkspaces",
		"CreateWorkspaceServiceAccount",
		"UpdatePermissions",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := grafanaOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(grafanaDataTypes) != 21 {
		t.Fatalf("expected 21 Grafana data types from docs, got %d", len(grafanaDataTypes))
	}
	if len(grafanaDataTypeByName) != len(grafanaDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"WorkspaceDescription",
		"WorkspaceSummary",
		"AuthenticationDescription",
		"ServiceAccountSummary",
		"PermissionEntry",
	}
	for _, typeName := range requiredTypes {
		if _, ok := grafanaDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestGrafanaStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := grafanaRequest(t, ts, http.MethodGet, "/grafana/UnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestGrafanaKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := grafanaRequest(t, ts, http.MethodGet, "/workspaces", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "workspaces") {
		t.Fatalf("expected ListWorkspaces response body to include workspaces, got %q", body)
	}
}

func TestGrafanaAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range grafanaOperations {
		path := grafanaPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := grafanaRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
