package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ivsChatMessagingRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "ivs-chat")
}

func TestIVSChatMessagingStage0CatalogCoverage(t *testing.T) {
	if len(ivsChatMessagingOperations) != 3 {
		t.Fatalf("expected 3 IVS Chat Messaging operations from docs, got %d", len(ivsChatMessagingOperations))
	}
	if len(ivsChatMessagingOperationByName) != len(ivsChatMessagingOperations) {
		t.Fatalf("expected unique IVS Chat Messaging operation names")
	}

	requiredActions := []string{
		"DeleteMessage",
		"DisconnectUser",
		"SendMessage",
	}
	for _, action := range requiredActions {
		if _, ok := ivsChatMessagingOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(ivsChatMessagingDataTypes) != 2 {
		t.Fatalf("expected 2 IVS Chat Messaging data types from docs, got %d", len(ivsChatMessagingDataTypes))
	}
	if len(ivsChatMessagingDataTypeByName) != len(ivsChatMessagingDataTypes) {
		t.Fatalf("expected unique IVS Chat Messaging data type names")
	}
}

func TestIVSChatMessagingStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ivsChatMessagingRequest(t, ts, http.MethodPost, "/NotARealIVSChatMessagingAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIVSChatMessagingStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ivsChatMessagingRequest(t, ts, http.MethodPost, "/SendMessage", `{"roomIdentifier":"room-00000001","content":"hello"}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "id") {
		t.Fatalf("expected SendMessage response body to include id, got %q", body)
	}
}

func TestIVSChatMessagingStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range ivsChatMessagingOperations {
		payload := `{}`
		resp := ivsChatMessagingRequest(t, ts, op.Method, op.URI, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
