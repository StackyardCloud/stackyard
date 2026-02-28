package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ssmGUIConnectRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "ssm-guiconnect")
}

func TestSSMGUIConnectStage0CatalogCoverage(t *testing.T) {
	if len(ssmGUIConnectOperations) != 3 {
		t.Fatalf("expected 3 SSM GUI Connect operations from docs, got %d", len(ssmGUIConnectOperations))
	}
	if len(ssmGUIConnectOperationByName) != len(ssmGUIConnectOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"DeleteConnectionRecordingPreferences",
		"GetConnectionRecordingPreferences",
		"UpdateConnectionRecordingPreferences",
	}
	for _, action := range requiredActions {
		if _, ok := ssmGUIConnectOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(ssmGUIConnectDataTypes) != 3 {
		t.Fatalf("expected 3 SSM GUI Connect data types from docs, got %d", len(ssmGUIConnectDataTypes))
	}
	if len(ssmGUIConnectDataTypeByName) != len(ssmGUIConnectDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ConnectionRecordingPreferences",
		"RecordingDestinations",
		"S3Bucket",
	}
	for _, typeName := range requiredTypes {
		if _, ok := ssmGUIConnectDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestSSMGUIConnectStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ssmGUIConnectRequest(t, ts, http.MethodPost, "/UnknownConnectionRecordingPreferences", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSSMGUIConnectKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ssmGUIConnectRequest(t, ts, http.MethodPost, "/GetConnectionRecordingPreferences", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "ConnectionRecordingPreferences") {
		t.Fatalf("expected GetConnectionRecordingPreferences response body to include ConnectionRecordingPreferences, got %q", body)
	}
}

func TestSSMGUIConnectAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range ssmGUIConnectOperations {
		resp := ssmGUIConnectRequest(t, ts, op.Method, op.URI, []byte(`{}`))
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
