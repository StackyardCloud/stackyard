package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func ivsChatRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "ivs-chat")
}

func ivsChatPathForOperation(op ivsChatOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{resourceArn}": url.PathEscape("arn:aws:ivs-chat:us-east-1:123456789012:room/room-00000001"),
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestIVSChatStage0CatalogCoverage(t *testing.T) {
	if len(ivsChatOperations) != 17 {
		t.Fatalf("expected 17 IVS Chat operations from docs, got %d", len(ivsChatOperations))
	}
	if len(ivsChatOperationByName) != len(ivsChatOperations) {
		t.Fatalf("expected unique IVS Chat operation names")
	}

	requiredActions := []string{
		"CreateRoom",
		"GetRoom",
		"ListRooms",
		"UpdateRoom",
		"DeleteRoom",
		"CreateLoggingConfiguration",
		"GetLoggingConfiguration",
		"ListLoggingConfigurations",
		"UpdateLoggingConfiguration",
		"DeleteLoggingConfiguration",
		"CreateChatToken",
		"SendEvent",
		"DisconnectUser",
		"DeleteMessage",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := ivsChatOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(ivsChatDataTypes) != 8 {
		t.Fatalf("expected 8 IVS Chat data types from docs, got %d", len(ivsChatDataTypes))
	}
	if len(ivsChatDataTypeByName) != len(ivsChatDataTypes) {
		t.Fatalf("expected unique IVS Chat data type names")
	}
}

func TestIVSChatStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ivsChatRequest(t, ts, http.MethodPost, "/NotARealIVSChatAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIVSChatStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ivsChatRequest(t, ts, http.MethodPost, "/ListRooms", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "rooms") {
		t.Fatalf("expected ListRooms response to include rooms, got %q", body)
	}
}

func TestIVSChatStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range ivsChatOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := ivsChatRequest(t, ts, op.Method, ivsChatPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
