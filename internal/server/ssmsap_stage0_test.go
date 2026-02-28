package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func ssmSAPRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "ssm-sap")
}

func ssmSAPPathForTest(template string) string {
	resourceARN := "arn:aws:ssm-sap:us-east-1:123456789012:application/app-000001"
	out := template
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestSSMSAPStage0CatalogCoverage(t *testing.T) {
	if len(ssmSAPOperations) != 27 {
		t.Fatalf("expected 27 SSM SAP operations from docs, got %d", len(ssmSAPOperations))
	}
	if len(ssmSAPOperationByName) != len(ssmSAPOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"RegisterApplication",
		"GetApplication",
		"ListApplications",
		"StartConfigurationChecks",
		"PutResourcePermission",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := ssmSAPOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(ssmSAPDataTypes) != 23 {
		t.Fatalf("expected 23 SSM SAP data types from docs, got %d", len(ssmSAPDataTypes))
	}
	if len(ssmSAPDataTypeByName) != len(ssmSAPDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Application",
		"Component",
		"ConfigurationCheckOperation",
		"Operation",
		"SubCheckResult",
	}
	for _, typeName := range requiredTypes {
		if _, ok := ssmSAPDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestSSMSAPStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ssmSAPRequest(t, ts, http.MethodPost, "/unknown-ssmsap-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSSMSAPKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ssmSAPRequest(t, ts, http.MethodPost, "/list-applications", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Applications") {
		t.Fatalf("expected ListApplications response body to include Applications, got %q", body)
	}
}

func TestSSMSAPAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range ssmSAPOperations {
		path := ssmSAPPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
			body = []byte(`{}`)
		}
		resp := ssmSAPRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
