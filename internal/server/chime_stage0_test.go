package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func chimeRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "chime")
}

func TestChimeStage0CatalogCoverage(t *testing.T) {
	if len(chimeOperations) != 62 {
		t.Fatalf("expected 62 Chime operations from docs, got %d", len(chimeOperations))
	}
	if len(chimeOperationByName) != len(chimeOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateAccount",
		"GetAccount",
		"ListAccounts",
		"CreateUser",
		"ListUsers",
		"UpdateUserSettings",
		"SearchAvailablePhoneNumbers",
	}
	for _, action := range requiredActions {
		if _, ok := chimeOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(chimeTypes) != 31 {
		t.Fatalf("expected 31 Chime data types from docs, got %d", len(chimeTypes))
	}
	if len(chimeTypeByName) != len(chimeTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Account",
		"AccountSettings",
		"Bot",
		"PhoneNumber",
		"Room",
		"User",
	}
	for _, typeName := range requiredTypes {
		if _, ok := chimeTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestChimeStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := chimeRequest(t, ts, http.MethodPost, "/chime-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestChimeStage0KnownActionReturnsListAccounts(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := chimeRequest(t, ts, http.MethodGet, "/accounts", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Accounts") {
		t.Fatalf("expected ListAccounts response body to include Accounts, got %q", body)
	}
}

func TestChimeStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"AccountId":          "acc-000001",
		"UserId":             "user-000001",
		"BotId":              "bot-000001",
		"PhoneNumberId":      "phone-number-000001",
		"PhoneNumberOrderId": "order-000001",
		"MeetingId":          "meeting-000001",
		"RoomId":             "room-000001",
		"MemberId":           "member-000001",
		"ConversationId":     "conversation-000001",
		"MessageId":          "message-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range chimeOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := chimeRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
