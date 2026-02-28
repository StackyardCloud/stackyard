package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func b2biRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	if strings.TrimSpace(payload) == "" {
		payload = `{}`
	}
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "B2BI." + action,
		},
		"b2bi",
	)
}

func TestB2BIStage0CatalogCoverage(t *testing.T) {
	if len(b2biOperations) != 30 {
		t.Fatalf("expected 30 B2BI operations from docs, got %d", len(b2biOperations))
	}
	if len(b2biOperationByName) != len(b2biOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateCapability",
		"CreatePartnership",
		"CreateProfile",
		"CreateTransformer",
		"StartTransformerJob",
		"TagResource",
		"ListTagsForResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := b2biOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(b2biDataTypes) != 43 {
		t.Fatalf("expected 43 B2BI data types from docs, got %d", len(b2biDataTypes))
	}
	if len(b2biDataTypeByName) != len(b2biDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"CapabilitySummary",
		"PartnershipSummary",
		"ProfileSummary",
		"TransformerSummary",
		"UpdateTransformer",
		"X12ValidationRule",
	}
	for _, typeName := range requiredTypes {
		if _, ok := b2biDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestB2BIStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := b2biRequest(t, ts, "UnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestB2BIStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := b2biRequest(t, ts, "ListCapabilities", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "capabilities") {
		t.Fatalf("expected ListCapabilities response body to include capabilities, got %q", body)
	}
}

func TestB2BIStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range b2biOperations {
		resp := b2biRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
