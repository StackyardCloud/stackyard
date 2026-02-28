package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func workmailRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "WorkMailService." + action,
		},
		"workmail",
	)
}

func TestWorkMailStage0CatalogCoverage(t *testing.T) {
	if len(workmailOperations) != 92 {
		t.Fatalf("expected 92 WorkMail operations from docs, got %d", len(workmailOperations))
	}
	if len(workmailOperationByName) != len(workmailOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateOrganization",
		"ListOrganizations",
		"CreateUser",
		"CreateGroup",
		"PutMailboxPermissions",
		"StartMailboxExportJob",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := workmailOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(workmailDataTypes) != 33 {
		t.Fatalf("expected 33 WorkMail data types from docs, got %d", len(workmailDataTypes))
	}
	if len(workmailDataTypeByName) != len(workmailDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"OrganizationSummary",
		"User",
		"Group",
		"Resource",
		"AccessControlRule",
		"MailboxExportJob",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := workmailDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestWorkMailStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workmailRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestWorkMailStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workmailRequest(t, ts, "ListOrganizations", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "OrganizationSummaries") {
		t.Fatalf("expected ListOrganizations response body to include OrganizationSummaries, got %q", body)
	}
}

func TestWorkMailStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range workmailOperations {
		resp := workmailRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
