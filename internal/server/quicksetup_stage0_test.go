package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func quickSetupRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "ssm-quicksetup")
}

func quickSetupPathForTest(template string) string {
	managerARN := "arn:aws:ssm-quicksetup:us-east-1:123456789012:configuration-manager/cm-000001"
	resourceARN := managerARN
	out := template
	out = strings.ReplaceAll(out, "{ManagerArn}", url.PathEscape(managerARN))
	out = strings.ReplaceAll(out, "{ConfigurationId}", "cfg-000001")
	out = strings.ReplaceAll(out, "{ResourceArn}", url.PathEscape(resourceARN))
	out = strings.ReplaceAll(out, "{Id}", "def-000001")
	return out
}

func TestQuickSetupStage0CatalogCoverage(t *testing.T) {
	if len(quickSetupOperations) != 14 {
		t.Fatalf("expected 14 Quick Setup operations from docs, got %d", len(quickSetupOperations))
	}
	if len(quickSetupOperationByName) != len(quickSetupOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateConfigurationManager",
		"GetServiceSettings",
		"ListConfigurationManagers",
		"ListConfigurations",
		"ListQuickSetupTypes",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := quickSetupOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(quickSetupDataTypes) != 10 {
		t.Fatalf("expected 10 Quick Setup data types from docs, got %d", len(quickSetupDataTypes))
	}
	if len(quickSetupDataTypeByName) != len(quickSetupDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ConfigurationDefinition",
		"ConfigurationManagerSummary",
		"QuickSetupTypeOutput",
		"ServiceSettings",
		"TagEntry",
	}
	for _, typeName := range requiredTypes {
		if _, ok := quickSetupDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestQuickSetupStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := quickSetupRequest(t, ts, http.MethodGet, "/serviceSettings/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestQuickSetupKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := quickSetupRequest(t, ts, http.MethodGet, "/listQuickSetupTypes", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "QuickSetupTypeList") {
		t.Fatalf("expected ListQuickSetupTypes response body to include QuickSetupTypeList, got %q", body)
	}
}

func TestQuickSetupAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range quickSetupOperations {
		path := quickSetupPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := quickSetupRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
