package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func systemsManagerRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AmazonSSM." + action,
		},
		"ssm",
	)
}

func TestSystemsManagerStage0CatalogCoverage(t *testing.T) {
	if len(systemsManagerOperations) != 146 {
		t.Fatalf("expected 146 Systems Manager operations from docs, got %d", len(systemsManagerOperations))
	}
	if len(systemsManagerOperationByName) != len(systemsManagerOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"ListDocuments",
		"DescribeDocument",
		"SendCommand",
		"StartSession",
		"GetServiceSetting",
		"UpdateServiceSetting",
	}
	for _, action := range requiredActions {
		if _, ok := systemsManagerOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(systemsManagerDataTypes) != 170 {
		t.Fatalf("expected 170 Systems Manager data types from docs, got %d", len(systemsManagerDataTypes))
	}
	if len(systemsManagerDataTypeByName) != len(systemsManagerDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Command",
		"DocumentDescription",
		"OpsItem",
		"Parameter",
		"Session",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := systemsManagerDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestSystemsManagerStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := systemsManagerRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSystemsManagerKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := systemsManagerRequest(t, ts, "ListDocuments", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "DocumentIdentifiers") {
		t.Fatalf("expected ListDocuments response body to include DocumentIdentifiers, got %q", body)
	}
}

func TestSystemsManagerStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range systemsManagerOperations {
		resp := systemsManagerRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
