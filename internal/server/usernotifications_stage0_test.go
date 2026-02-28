package server

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func userNotificationsRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"notifications",
	)
}

func userNotificationsConcretePath(path string) string {
	re := regexp.MustCompile(`\{[^}]+\}`)
	return re.ReplaceAllString(path, "stackyard-id")
}

func TestUserNotificationsStage0CatalogCoverage(t *testing.T) {
	if len(userNotificationsOperations) != 39 {
		t.Fatalf("expected 39 User Notifications operations from docs, got %d", len(userNotificationsOperations))
	}
	if len(userNotificationsOperationByName) != len(userNotificationsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateNotificationConfiguration",
		"ListNotificationEvents",
		"TagResource",
		"RegisterNotificationHub",
	}
	for _, action := range requiredActions {
		if _, ok := userNotificationsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(userNotificationsDataTypes) != 34 {
		t.Fatalf("expected 34 User Notifications data types from docs, got %d", len(userNotificationsDataTypes))
	}
	if len(userNotificationsDataTypeByName) != len(userNotificationsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"NotificationConfigurationStructure",
		"NotificationEvent",
		"NotificationHubOverview",
	}
	for _, typeName := range requiredTypes {
		if _, ok := userNotificationsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestUserNotificationsStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := userNotificationsRequest(t, ts, http.MethodGet, "/totally-unknown-action", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestUserNotificationsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := userNotificationsRequest(t, ts, http.MethodGet, "/notification-configurations", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "notificationConfigurations") {
		t.Fatalf("expected ListNotificationConfigurations response body to include notificationConfigurations, got %q", body)
	}
}

func TestUserNotificationsStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range userNotificationsOperations {
		payload := ""
		if op.Method == http.MethodPost || op.Method == http.MethodPut {
			payload = `{}`
		}
		resp := userNotificationsRequest(t, ts, op.Method, userNotificationsConcretePath(op.URI), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
