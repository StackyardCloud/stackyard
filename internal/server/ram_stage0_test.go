package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ramRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSRamFrontEndService." + action,
		},
		"ram",
	)
}

func TestRAMStage0CatalogCoverage(t *testing.T) {
	if len(ramOperations) != 35 {
		t.Fatalf("expected 35 RAM operations from docs, got %d", len(ramOperations))
	}
	if len(ramOperationByName) != len(ramOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateResourceShare",
		"GetResourceShares",
		"AssociateResourceShare",
		"TagResource",
		"UntagResource",
		"ListResourceTypes",
	}
	for _, action := range requiredActions {
		if _, ok := ramOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(ramDataTypes) != 14 {
		t.Fatalf("expected 14 RAM data types from docs, got %d", len(ramDataTypes))
	}
	if len(ramDataTypeByName) != len(ramDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Principal",
		"Resource",
		"ResourceShare",
		"ResourceShareAssociation",
		"ResourceShareInvitation",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := ramDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestRAMStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ramRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestRAMKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ramRequest(t, ts, "GetResourceShares", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "resourceShares") {
		t.Fatalf("expected GetResourceShares response body to include resourceShares, got %q", body)
	}
}

func TestRAMStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range ramOperations {
		resp := ramRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
