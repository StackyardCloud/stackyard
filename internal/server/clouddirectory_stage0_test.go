package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cloudDirectoryRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "clouddirectory")
}

func TestCloudDirectoryStage0CatalogCoverage(t *testing.T) {
	if len(cloudDirectoryOperations) != 66 {
		t.Fatalf("expected 66 Cloud Directory operations from docs, got %d", len(cloudDirectoryOperations))
	}
	if len(cloudDirectoryOperationByName) != len(cloudDirectoryOperations) {
		t.Fatalf("expected unique Cloud Directory operation names")
	}

	requiredActions := []string{
		"CreateDirectory",
		"GetDirectory",
		"ListDirectories",
		"CreateSchema",
		"CreateObject",
		"BatchRead",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := cloudDirectoryOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(cloudDirectoryAPITypes) != 96 {
		t.Fatalf("expected 96 Cloud Directory data types from docs, got %d", len(cloudDirectoryAPITypes))
	}
	if len(cloudDirectoryAPITypeByName) != len(cloudDirectoryAPITypes) {
		t.Fatalf("expected unique Cloud Directory data type names")
	}
}

func TestCloudDirectoryStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudDirectoryRequest(t, ts, http.MethodPut, "/amazonclouddirectory/2017-01-11/not-a-real-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCloudDirectoryStage0KnownRouteReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudDirectoryRequest(t, ts, http.MethodPost, "/amazonclouddirectory/2017-01-11/directory/list", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Directories") {
		t.Fatalf("expected ListDirectories response body to include Directories, got %q", body)
	}
}

func TestCloudDirectoryStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cloudDirectoryOperations {
		resp := cloudDirectoryRequest(t, ts, op.Method, op.URI, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
