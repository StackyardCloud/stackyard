package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func keyspacesRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "KeyspacesService." + action,
		},
		"cassandra",
	)
}

func TestKeyspacesStage0OperationCoverage(t *testing.T) {
	if len(keyspacesOperations) != 19 {
		t.Fatalf("expected 19 Keyspaces operations from docs, got %d", len(keyspacesOperations))
	}
	if len(keyspacesOperationByName) != len(keyspacesOperations) {
		t.Fatalf("expected unique operation names")
	}

	required := []string{
		"CreateKeyspace",
		"CreateTable",
		"GetTable",
		"ListKeyspaces",
		"UpdateTable",
		"GetType",
		"CreateType",
		"DeleteType",
	}
	for _, name := range required {
		if _, ok := keyspacesOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestKeyspacesStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := keyspacesRequest(t, ts, "TotallyUnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestKeyspacesStage0KnownActionDoesNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := keyspacesRequest(t, ts, "CreateType", []byte(`{"keyspaceName":"demo","typeName":"my_type","fieldDefinitions":[{"name":"f1","type":"text"}]}`))
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected CreateType to be implemented")
	}
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected non-NotImplemented response body, got %q", body)
	}
}

func TestKeyspacesStage0KnownImplementedActionDoesNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := keyspacesRequest(t, ts, "ListKeyspaces", []byte(`{"maxResults":10}`))
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected ListKeyspaces to be implemented")
	}
}
