package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQDeveloperStage0CatalogCoverage(t *testing.T) {
	if len(qDeveloperOperations) != 34 {
		t.Fatalf("expected 34 AWS Q Developer operations from docs, got %d", len(qDeveloperOperations))
	}
	if len(qDeveloperOperationByName) != len(qDeveloperOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateSlackChannelConfiguration",
		"CreateCustomAction",
		"DescribeSlackWorkspaces",
		"ListAssociations",
		"GetAccountPreferences",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := qDeveloperOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(qDeveloperDataTypes) != 15 {
		t.Fatalf("expected 15 AWS Q Developer data types from docs, got %d", len(qDeveloperDataTypes))
	}
	if len(qDeveloperDataTypeByName) != len(qDeveloperDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AccountPreferences",
		"AssociationListing",
		"CustomAction",
		"SlackChannelConfiguration",
		"TeamsChannelConfiguration",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := qDeveloperDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func qDeveloperRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
	}
	headers := map[string]string{}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "chatbot")
}

func TestQDeveloperStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := qDeveloperRequest(t, ts, http.MethodPost, "/unknown-qdeveloper-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestQDeveloperKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := qDeveloperRequest(t, ts, http.MethodPost, "/get-account-preferences", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "AccountPreferences") {
		t.Fatalf("expected GetAccountPreferences response body to include AccountPreferences, got %q", body)
	}
}

func TestQDeveloperStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range qDeveloperOperations {
		resp := qDeveloperRequest(t, ts, op.Method, op.URI, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
