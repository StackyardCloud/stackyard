package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func controlCatalogRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "controlcatalog")
}

func TestControlCatalogStage0CatalogCoverage(t *testing.T) {
	if len(controlCatalogOperations) != 6 {
		t.Fatalf("expected 6 Control Catalog operations from docs, got %d", len(controlCatalogOperations))
	}
	if len(controlCatalogOperationByName) != len(controlCatalogOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"GetControl",
		"ListCommonControls",
		"ListControlMappings",
		"ListControls",
		"ListDomains",
		"ListObjectives",
	}
	for _, action := range requiredActions {
		if _, ok := controlCatalogOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(controlCatalogDataTypes) != 23 {
		t.Fatalf("expected 23 Control Catalog data types from docs, got %d", len(controlCatalogDataTypes))
	}
	if len(controlCatalogDataTypeByName) != len(controlCatalogDataTypes) {
		t.Fatalf("expected unique data type names")
	}
}

func TestControlCatalogUnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := controlCatalogRequest(t, ts, http.MethodPost, "/controlcatalog-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestControlCatalogAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range controlCatalogOperations {
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := controlCatalogRequest(t, ts, op.Method, op.URI, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}

func TestControlCatalogKnownActionReturnsControl(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := controlCatalogRequest(t, ts, http.MethodPost, "/get-control", []byte(`{"ControlArn":"arn:aws:controlcatalog:us-east-1::control/cis-1-1"}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "Control") {
		t.Fatalf("expected GetControl response to include Control, got %q", body)
	}
}
