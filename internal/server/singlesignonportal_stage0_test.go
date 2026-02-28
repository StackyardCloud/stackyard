package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func singleSignOnPortalRequest(t *testing.T, ts *httptest.Server, method, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, ts.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("x-amz-sso_bearer_token", "stackyard-access-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestSingleSignOnPortalStage0CatalogCoverage(t *testing.T) {
	if len(singleSignOnPortalOperations) != 4 {
		t.Fatalf("expected 4 IAM Identity Center Portal operations from docs, got %d", len(singleSignOnPortalOperations))
	}
	if len(singleSignOnPortalOperationByName) != len(singleSignOnPortalOperations) {
		t.Fatalf("expected unique IAM Identity Center Portal operation names")
	}
	requiredActions := []string{
		"GetRoleCredentials",
		"ListAccountRoles",
		"ListAccounts",
		"Logout",
	}
	for _, action := range requiredActions {
		if _, ok := singleSignOnPortalOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(singleSignOnPortalDataTypes) != 3 {
		t.Fatalf("expected 3 IAM Identity Center Portal data types from docs, got %d", len(singleSignOnPortalDataTypes))
	}
	if len(singleSignOnPortalDataTypeByName) != len(singleSignOnPortalDataTypes) {
		t.Fatalf("expected unique IAM Identity Center Portal data type names")
	}
}

func TestSingleSignOnPortalStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := singleSignOnPortalRequest(t, ts, http.MethodGet, "/assignment/unknown")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSingleSignOnPortalStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	cases := []struct {
		action string
		method string
		path   string
	}{
		{action: "ListAccounts", method: http.MethodGet, path: "/assignment/accounts"},
		{action: "ListAccountRoles", method: http.MethodGet, path: "/assignment/roles?account_id=123456789012"},
		{action: "GetRoleCredentials", method: http.MethodGet, path: "/federation/credentials?account_id=123456789012&role_name=stackyard-role"},
		{action: "Logout", method: http.MethodPost, path: "/logout"},
	}

	for _, tc := range cases {
		resp := singleSignOnPortalRequest(t, ts, tc.method, tc.path)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", tc.action, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", tc.action, resp.StatusCode, body)
		}
	}
}
