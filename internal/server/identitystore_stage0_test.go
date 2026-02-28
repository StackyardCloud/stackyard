package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func identityStoreRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
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
			"X-Amz-Target": "AWSIdentityStore." + action,
		},
		"identitystore",
	)
}

func TestIdentityStoreStage0CatalogCoverage(t *testing.T) {
	if len(identityStoreOperations) != 19 {
		t.Fatalf("expected 19 Identity Store operations from docs, got %d", len(identityStoreOperations))
	}
	if len(identityStoreOperationByName) != len(identityStoreOperations) {
		t.Fatalf("expected unique Identity Store operation names")
	}

	requiredActions := []string{
		"CreateUser",
		"CreateGroup",
		"CreateGroupMembership",
		"GetUserId",
		"GetGroupId",
		"IsMemberInGroups",
	}
	for _, action := range requiredActions {
		if _, ok := identityStoreOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(identityStoreDataTypes) != 17 {
		t.Fatalf("expected 17 Identity Store data types from docs, got %d", len(identityStoreDataTypes))
	}
	if len(identityStoreDataTypeByName) != len(identityStoreDataTypes) {
		t.Fatalf("expected unique Identity Store data type names")
	}
}

func TestIdentityStoreStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := identityStoreRequest(t, ts, "DefinitelyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIdentityStoreStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range identityStoreOperations {
		resp := identityStoreRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
