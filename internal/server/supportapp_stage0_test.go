package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func supportAppRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"supportapp",
	)
}

func TestSupportAppStage0CatalogCoverage(t *testing.T) {
	if len(supportAppOperations) != 10 {
		t.Fatalf("expected 10 Support App operations from docs, got %d", len(supportAppOperations))
	}
	if len(supportAppOperationByName) != len(supportAppOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateSlackChannelConfiguration",
		"GetAccountAlias",
		"ListSlackChannelConfigurations",
		"ListSlackWorkspaceConfigurations",
		"UpdateSlackChannelConfiguration",
	}
	for _, action := range requiredActions {
		if _, ok := supportAppOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(supportAppDataTypes) != 2 {
		t.Fatalf("expected 2 Support App data types from docs, got %d", len(supportAppDataTypes))
	}
	if len(supportAppDataTypeByName) != len(supportAppDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"SlackChannelConfiguration",
		"SlackWorkspaceConfiguration",
	}
	for _, typeName := range requiredTypes {
		if _, ok := supportAppDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestSupportAppStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := supportAppRequest(t, ts, http.MethodPost, "/control/totally-unknown-action", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSupportAppKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := supportAppRequest(t, ts, http.MethodPost, "/control/list-slack-workspace-configurations", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "slackWorkspaceConfigurations") {
		t.Fatalf("expected ListSlackWorkspaceConfigurations response body to include slackWorkspaceConfigurations, got %q", body)
	}
}

func TestSupportAppStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range supportAppOperations {
		resp := supportAppRequest(t, ts, op.Method, op.URI, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
