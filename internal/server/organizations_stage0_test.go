package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOrganizationsStage0CatalogCoverage(t *testing.T) {
	if len(organizationsOperations) != 63 {
		t.Fatalf("expected 63 Organizations operations from docs, got %d", len(organizationsOperations))
	}
	if len(organizationsOperationByName) != len(organizationsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateOrganization",
		"DescribeOrganization",
		"ListAccounts",
		"InviteAccountToOrganization",
		"UpdatePolicy",
		"PutResourcePolicy",
	}
	for _, action := range requiredActions {
		if _, ok := organizationsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(organizationsDataTypes) != 25 {
		t.Fatalf("expected 25 Organizations data types from docs, got %d", len(organizationsDataTypes))
	}
	if len(organizationsDataTypeByName) != len(organizationsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Organization",
		"OrganizationalUnit",
		"Policy",
		"ResourcePolicy",
		"ResponsibilityTransfer",
		"TransferParticipant",
	}
	for _, typeName := range requiredTypes {
		if _, ok := organizationsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func organizationsRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSOrganizationsV20161128." + action,
		},
		"organizations",
	)
}

func TestOrganizationsStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := organizationsRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestOrganizationsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := organizationsRequest(t, ts, "DescribeOrganization", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "Organization") {
		t.Fatalf("expected DescribeOrganization response body to include Organization, got %q", body)
	}
}

func TestOrganizationsStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range organizationsOperations {
		resp := organizationsRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
