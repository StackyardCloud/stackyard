package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSTSStage0CatalogCoverage(t *testing.T) {
	if len(stsOperations) != 12 {
		t.Fatalf("expected 12 STS operations from docs, got %d", len(stsOperations))
	}
	if len(stsOperationByName) != len(stsOperations) {
		t.Fatalf("expected unique STS operation names")
	}

	requiredActions := []string{
		"AssumeRole",
		"AssumeRoleWithSAML",
		"AssumeRoleWithWebIdentity",
		"DecodeAuthorizationMessage",
		"GetCallerIdentity",
		"GetFederationToken",
		"GetSessionToken",
	}
	for _, action := range requiredActions {
		if _, ok := stsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(stsDataTypes) != 8 {
		t.Fatalf("expected 8 STS data types from docs, got %d", len(stsDataTypes))
	}
	if len(stsDataTypeByName) != len(stsDataTypes) {
		t.Fatalf("expected unique STS data type names")
	}
}

func stsRequest(t *testing.T, ts *httptest.Server, action string, params url.Values) *http.Response {
	t.Helper()
	if params == nil {
		params = url.Values{}
	}
	params.Set("Action", action)
	params.Set("Version", "2011-06-15")
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(params.Encode()),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		"sts",
	)
}

func TestSTSStage0UnknownActionReturnsInvalidAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := stsRequest(t, ts, "DefinitelyUnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "InvalidAction") {
		t.Fatalf("expected InvalidAction response body, got %q", body)
	}
}

func TestSTSStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := stsRequest(t, ts, "GetCallerIdentity", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "GetCallerIdentityResponse") {
		t.Fatalf("expected GetCallerIdentityResponse in body, got %q", body)
	}
}

func TestSTSStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range stsOperations {
		resp := stsRequest(t, ts, op.Name, nil)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
