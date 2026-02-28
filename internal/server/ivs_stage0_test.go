package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func ivsRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "ivs")
}

func ivsPathForOperation(op ivsOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{resourceArn}": url.PathEscape("arn:aws:ivs:us-east-1:123456789012:channel/channel-00000001"),
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestIVSStage0CatalogCoverage(t *testing.T) {
	if len(ivsOperations) != 35 {
		t.Fatalf("expected 35 IVS operations from docs, got %d", len(ivsOperations))
	}
	if len(ivsOperationByName) != len(ivsOperations) {
		t.Fatalf("expected unique IVS operation names")
	}

	requiredActions := []string{
		"CreateChannel",
		"GetChannel",
		"ListChannels",
		"UpdateChannel",
		"DeleteChannel",
		"CreateStreamKey",
		"GetStream",
		"StopStream",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := ivsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(ivsDataTypes) != 30 {
		t.Fatalf("expected 30 IVS data types from docs, got %d", len(ivsDataTypes))
	}
	if len(ivsDataTypeByName) != len(ivsDataTypes) {
		t.Fatalf("expected unique IVS data type names")
	}
}

func TestIVSStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ivsRequest(t, ts, http.MethodPost, "/NotARealIVSAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIVSStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ivsRequest(t, ts, http.MethodPost, "/ListChannels", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "channels") {
		t.Fatalf("expected ListChannels response to include channels, got %q", body)
	}
}

func TestIVSStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range ivsOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := ivsRequest(t, ts, op.Method, ivsPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
