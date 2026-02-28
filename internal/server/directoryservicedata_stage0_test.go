package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func directoryServiceDataRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "ds-data")
}

func TestDirectoryServiceDataStage0CatalogCoverage(t *testing.T) {
	if len(directoryServiceDataOperations) != 17 {
		t.Fatalf("expected 17 Directory Service Data operations from docs, got %d", len(directoryServiceDataOperations))
	}
	if len(directoryServiceDataOperationByName) != len(directoryServiceDataOperations) {
		t.Fatalf("expected unique Directory Service Data operation names")
	}

	requiredActions := []string{
		"CreateUser",
		"UpdateUser",
		"DeleteUser",
		"DescribeUser",
		"ListUsers",
		"CreateGroup",
		"AddGroupMember",
		"ListGroupsForMember",
	}
	for _, action := range requiredActions {
		if _, ok := directoryServiceDataOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(directoryServiceDataAPITypes) != 47 {
		t.Fatalf("expected 47 Directory Service Data data types from docs, got %d", len(directoryServiceDataAPITypes))
	}
	if len(directoryServiceDataAPITypeByName) != len(directoryServiceDataAPITypes) {
		t.Fatalf("expected unique Directory Service Data data type names")
	}

	requiredTypes := []string{
		"AddGroupMemberRequest",
		"CreateUserRequest",
		"DescribeUserResult",
		"ListGroupsResult",
		"SearchUsersResult",
		"ValidationException",
	}
	for _, typeName := range requiredTypes {
		if _, ok := directoryServiceDataAPITypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDirectoryServiceDataStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := directoryServiceDataRequest(t, ts, http.MethodPost, "/NotARealDirectoryServiceDataAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDirectoryServiceDataStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := directoryServiceDataRequest(t, ts, http.MethodPost, "/Users/ListUsers", `{"DirectoryId":"d-0000000000","MaxResults":10}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Users") {
		t.Fatalf("expected ListUsers response body to include Users, got %q", body)
	}
}

func TestDirectoryServiceDataStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range directoryServiceDataOperations {
		resp := directoryServiceDataRequest(t, ts, op.Method, op.URI, `{"DirectoryId":"d-0000000000"}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
