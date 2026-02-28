package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func ivsRealtimeRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "ivs-realtime")
}

func ivsRealtimePathForOperation(op ivsRealtimeOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{resourceArn}": url.PathEscape("arn:aws:ivs-realtime:us-east-1:123456789012:stage/stage-00000001"),
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestIVSRealtimeStage0CatalogCoverage(t *testing.T) {
	if len(ivsRealtimeOperations) != 39 {
		t.Fatalf("expected 39 IVS Real-Time operations from docs, got %d", len(ivsRealtimeOperations))
	}
	if len(ivsRealtimeOperationByName) != len(ivsRealtimeOperations) {
		t.Fatalf("expected unique IVS Real-Time operation names")
	}

	requiredActions := []string{
		"CreateStage",
		"GetStage",
		"ListStages",
		"UpdateStage",
		"DeleteStage",
		"CreateParticipantToken",
		"GetParticipant",
		"ListParticipants",
		"StartComposition",
		"StopComposition",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := ivsRealtimeOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(ivsRealtimeDataTypes) != 40 {
		t.Fatalf("expected 40 IVS Real-Time data types from docs, got %d", len(ivsRealtimeDataTypes))
	}
	if len(ivsRealtimeDataTypeByName) != len(ivsRealtimeDataTypes) {
		t.Fatalf("expected unique IVS Real-Time data type names")
	}
}

func TestIVSRealtimeStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ivsRealtimeRequest(t, ts, http.MethodPost, "/NotARealIVSRealtimeAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIVSRealtimeStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ivsRealtimeRequest(t, ts, http.MethodPost, "/ListStages", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "stages") {
		t.Fatalf("expected ListStages response to include stages, got %q", body)
	}
}

func TestIVSRealtimeStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range ivsRealtimeOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := ivsRealtimeRequest(t, ts, op.Method, ivsRealtimePathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
