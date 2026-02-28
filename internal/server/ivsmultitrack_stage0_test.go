package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ivsMultitrackRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "ivs")
}

func ivsMultitrackPathForOperation(op ivsMultitrackOperation) string {
	return op.URI
}

func TestIVSMultitrackStage0CatalogCoverage(t *testing.T) {
	if len(ivsMultitrackOperations) != 2 {
		t.Fatalf("expected 2 IVS multitrack operations from docs, got %d", len(ivsMultitrackOperations))
	}
	if len(ivsMultitrackOperationByName) != len(ivsMultitrackOperations) {
		t.Fatalf("expected unique IVS multitrack operation names")
	}

	requiredActions := []string{"FindIngest", "GetClientConfiguration"}
	for _, action := range requiredActions {
		if _, ok := ivsMultitrackOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(ivsMultitrackDataTypes) != 19 {
		t.Fatalf("expected 19 IVS multitrack data types from docs, got %d", len(ivsMultitrackDataTypes))
	}
	if len(ivsMultitrackDataTypeByName) != len(ivsMultitrackDataTypes) {
		t.Fatalf("expected unique IVS multitrack data type names")
	}
}

func TestIVSMultitrackStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ivsMultitrackRequest(t, ts, http.MethodPost, "/api/v3/NotARealAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIVSMultitrackStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ivsMultitrackRequest(t, ts, http.MethodGet, "/api/v2/FindIngest", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "ingest") {
		t.Fatalf("expected FindIngest response to include ingest, got %q", body)
	}
}

func TestIVSMultitrackStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range ivsMultitrackOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := ivsMultitrackRequest(t, ts, op.Method, ivsMultitrackPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
