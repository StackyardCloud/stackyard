package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func managedServicesCMRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSManagedServicesChangeManagement." + action,
		},
		"amscm",
	)
}

func TestManagedServicesCMStage0CatalogCoverage(t *testing.T) {
	if len(managedServicesCMOperations) != 22 {
		t.Fatalf("expected 22 Managed Services CM actions from docs, got %d", len(managedServicesCMOperations))
	}
	if len(managedServicesCMOperationByName) != len(managedServicesCMOperations) {
		t.Fatalf("expected unique Managed Services CM action names")
	}

	requiredActions := []string{
		"CreateRfc",
		"GetRfc",
		"UpdateRfc",
		"SubmitRfc",
		"ApproveRfc",
		"ListRfcSummaries",
		"ListRestrictedExecutionTimes",
	}
	for _, action := range requiredActions {
		if _, ok := managedServicesCMOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(managedServicesCMDataTypes) != 27 {
		t.Fatalf("expected 27 Managed Services CM data types from docs, got %d", len(managedServicesCMDataTypes))
	}
	if len(managedServicesCMDataTypeByName) != len(managedServicesCMDataTypes) {
		t.Fatalf("expected unique Managed Services CM data type names")
	}

	requiredTypes := []string{
		"Rfc",
		"RfcSummary",
		"RfcAttachment",
		"RfcAttachmentSummary",
		"RfcCorrespondence",
		"RestrictedExecutionTime",
	}
	for _, typeName := range requiredTypes {
		if _, ok := managedServicesCMDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestManagedServicesCMStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := managedServicesCMRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestManagedServicesCMStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := managedServicesCMRequest(t, ts, "ListRfcSummaries", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "RfcSummaries") {
		t.Fatalf("expected ListRfcSummaries response to include RfcSummaries, got %q", body)
	}
}

func TestManagedServicesCMStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range managedServicesCMOperations {
		resp := managedServicesCMRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
