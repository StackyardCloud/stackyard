package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func notificationsContactsRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"notificationscontacts",
	)
}

func notificationsContactsConcretePath(path string) string {
	encodedArn := url.PathEscape("arn:aws:notificationscontacts:us-east-1:123456789012:emailcontact/ec-000001")
	path = strings.ReplaceAll(path, "{arn}", encodedArn)
	path = strings.ReplaceAll(path, "{code}", "123456")
	return path
}

func TestNotificationsContactsStage0CatalogCoverage(t *testing.T) {
	if len(notificationsContactsOperations) != 9 {
		t.Fatalf("expected 9 Notifications Contacts operations from docs, got %d", len(notificationsContactsOperations))
	}
	if len(notificationsContactsOperationByName) != len(notificationsContactsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateEmailContact",
		"ListEmailContacts",
		"ActivateEmailContact",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := notificationsContactsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(notificationsContactsDataTypes) != 2 {
		t.Fatalf("expected 2 Notifications Contacts data types from docs, got %d", len(notificationsContactsDataTypes))
	}
	if len(notificationsContactsDataTypeByName) != len(notificationsContactsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{"EmailContact", "ValidationExceptionField"}
	for _, typeName := range requiredTypes {
		if _, ok := notificationsContactsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestNotificationsContactsStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := notificationsContactsRequest(t, ts, http.MethodGet, "/notificationscontacts-unknown", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestNotificationsContactsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := notificationsContactsRequest(t, ts, http.MethodGet, "/emailcontacts", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "emailContacts") {
		t.Fatalf("expected ListEmailContacts response body to include emailContacts, got %q", body)
	}
}

func TestNotificationsContactsStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range notificationsContactsOperations {
		payload := ""
		if op.Method == http.MethodPost || op.Method == http.MethodPut {
			payload = `{}`
		}
		resp := notificationsContactsRequest(t, ts, op.Method, notificationsContactsConcretePath(op.URI), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
