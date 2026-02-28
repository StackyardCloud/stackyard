package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func singleSignOnOIDCRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
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
			"X-Amz-Target": "AWSSSOOIDCService." + action,
		},
		"sso-oidc",
	)
}

func singleSignOnOIDCRestRequest(t *testing.T, ts *httptest.Server, path, payload string) *http.Response {
	t.Helper()
	if strings.TrimSpace(payload) == "" {
		payload = `{}`
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader([]byte(payload)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestSingleSignOnOIDCStage0CatalogCoverage(t *testing.T) {
	if len(singleSignOnOIDCOperations) != 4 {
		t.Fatalf("expected 4 IAM Identity Center OIDC operations from docs, got %d", len(singleSignOnOIDCOperations))
	}
	if len(singleSignOnOIDCOperationByName) != len(singleSignOnOIDCOperations) {
		t.Fatalf("expected unique IAM Identity Center OIDC operation names")
	}

	requiredActions := []string{
		"RegisterClient",
		"StartDeviceAuthorization",
		"CreateToken",
		"CreateTokenWithIAM",
	}
	for _, action := range requiredActions {
		if _, ok := singleSignOnOIDCOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(singleSignOnOIDCDataTypes) != 1 {
		t.Fatalf("expected 1 IAM Identity Center OIDC data type from docs, got %d", len(singleSignOnOIDCDataTypes))
	}
	if len(singleSignOnOIDCDataTypeByName) != len(singleSignOnOIDCDataTypes) {
		t.Fatalf("expected unique IAM Identity Center OIDC data type names")
	}
}

func TestSingleSignOnOIDCStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := singleSignOnOIDCRequest(t, ts, "DefinitelyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSingleSignOnOIDCStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range singleSignOnOIDCOperations {
		resp := singleSignOnOIDCRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}

func TestSingleSignOnOIDCStage0RestRoutes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := singleSignOnOIDCRestRequest(
		t,
		ts,
		"/client/register",
		`{"clientName":"stackyard-client","clientType":"public"}`,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = singleSignOnOIDCRestRequest(
		t,
		ts,
		"/device_authorization",
		`{"clientId":"stackyard-client-id","clientSecret":"stackyard-client-secret","startUrl":"https://stackyard.awsapps.com/start"}`,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = singleSignOnOIDCRestRequest(
		t,
		ts,
		"/token",
		`{"clientId":"stackyard-client-id","clientSecret":"stackyard-client-secret","grantType":"urn:ietf:params:oauth:grant-type:device_code","deviceCode":"stackyard-device-code"}`,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/token?aws_iam=t",
		[]byte(`{"clientId":"stackyard-client-id","grantType":"refresh_token","refreshToken":"stackyard-refresh-token"}`),
		map[string]string{"Content-Type": "application/json"},
		"sso-oauth",
	)
	assertStatus(t, resp, http.StatusOK)
}
