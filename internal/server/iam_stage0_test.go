package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestIAMStage0CatalogCoverage(t *testing.T) {
	if len(iamOperations) != 177 {
		t.Fatalf("expected 177 IAM operations from docs, got %d", len(iamOperations))
	}
	if len(iamOperationByName) != len(iamOperations) {
		t.Fatalf("expected unique IAM operation names")
	}

	requiredActions := []string{
		"ListUsers",
		"ListGroups",
		"ListRoles",
		"CreateUser",
		"CreateRole",
		"CreatePolicy",
		"GetAccountSummary",
		"SimulatePrincipalPolicy",
	}
	for _, action := range requiredActions {
		if _, ok := iamOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(iamDataTypes) != 57 {
		t.Fatalf("expected 57 IAM data types from docs, got %d", len(iamDataTypes))
	}
	if len(iamDataTypeByName) != len(iamDataTypes) {
		t.Fatalf("expected unique IAM data type names")
	}
}

func iamRequest(t *testing.T, ts *httptest.Server, action string, params url.Values) *http.Response {
	t.Helper()
	if params == nil {
		params = url.Values{}
	}
	params.Set("Action", action)
	params.Set("Version", "2010-05-08")
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(params.Encode()),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		"iam",
	)
}

func TestIAMStage0UnknownActionReturnsInvalidAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iamRequest(t, ts, "DefinitelyUnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "InvalidAction") {
		t.Fatalf("expected InvalidAction response body, got %q", body)
	}
}

func TestIAMStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iamRequest(t, ts, "ListUsers", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "ListUsersResponse") {
		t.Fatalf("expected ListUsersResponse in body, got %q", body)
	}
}

func TestIAMStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range iamOperations {
		resp := iamRequest(t, ts, op.Name, nil)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
