package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func singleSignOnRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	if strings.TrimSpace(payload) == "" {
		payload = `{}`
	}
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "SWBExternalService." + action,
		},
		"sso",
	)
}

func TestSingleSignOnStage0CatalogCoverage(t *testing.T) {
	if len(singleSignOnOperations) != 79 {
		t.Fatalf("expected 79 IAM Identity Center operations from docs, got %d", len(singleSignOnOperations))
	}
	if len(singleSignOnOperationByName) != len(singleSignOnOperations) {
		t.Fatalf("expected unique IAM Identity Center operation names")
	}

	requiredActions := []string{
		"CreateInstance",
		"ListInstances",
		"CreatePermissionSet",
		"ProvisionPermissionSet",
		"CreateApplication",
		"CreateTrustedTokenIssuer",
	}
	for _, action := range requiredActions {
		if _, ok := singleSignOnOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(singleSignOnDataTypes) != 49 {
		t.Fatalf("expected 49 IAM Identity Center data types from docs, got %d", len(singleSignOnDataTypes))
	}
	if len(singleSignOnDataTypeByName) != len(singleSignOnDataTypes) {
		t.Fatalf("expected unique IAM Identity Center data type names")
	}
}

func TestSingleSignOnStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := singleSignOnRequest(t, ts, "DefinitelyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSingleSignOnStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range singleSignOnOperations {
		resp := singleSignOnRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
