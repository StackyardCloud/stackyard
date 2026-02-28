package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func supportRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSSupport_20130415." + action,
		},
		"support",
	)
}

func TestSupportStage0CatalogCoverage(t *testing.T) {
	if len(supportOperations) != 16 {
		t.Fatalf("expected 16 Support operations from docs, got %d", len(supportOperations))
	}
	if len(supportOperationByName) != len(supportOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateCase",
		"DescribeCases",
		"DescribeServices",
		"DescribeTrustedAdvisorCheckResult",
		"RefreshTrustedAdvisorCheck",
		"ResolveCase",
	}
	for _, action := range requiredActions {
		if _, ok := supportOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(supportDataTypes) != 21 {
		t.Fatalf("expected 21 Support data types from docs, got %d", len(supportDataTypes))
	}
	if len(supportDataTypeByName) != len(supportDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Attachment",
		"CaseDetails",
		"Communication",
		"Service",
		"SeverityLevel",
	}
	for _, typeName := range requiredTypes {
		if _, ok := supportDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestSupportStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := supportRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSupportKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := supportRequest(t, ts, "DescribeServices", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "services") {
		t.Fatalf("expected DescribeServices response body to include services, got %q", body)
	}
}

func TestSupportStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range supportOperations {
		resp := supportRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
