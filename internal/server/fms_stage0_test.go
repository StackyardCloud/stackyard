package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fmsRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
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
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSFMS_20180101." + action,
		},
		"fms",
	)
}

func TestFMSStage0CatalogCoverage(t *testing.T) {
	if len(fmsOperations) != 42 {
		t.Fatalf("expected 42 FMS operations from docs, got %d", len(fmsOperations))
	}
	if len(fmsOperationByName) != len(fmsOperations) {
		t.Fatalf("expected unique FMS operation names")
	}

	requiredActions := []string{
		"PutPolicy",
		"GetPolicy",
		"ListPolicies",
		"PutResourceSet",
		"TagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := fmsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(fmsDataTypes) != 90 {
		t.Fatalf("expected 90 FMS data types from docs, got %d", len(fmsDataTypes))
	}
	if len(fmsDataTypeByName) != len(fmsDataTypes) {
		t.Fatalf("expected unique FMS data type names")
	}
}

func TestFMSStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := fmsRequest(t, ts, "NotARealAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestFMSStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := fmsRequest(t, ts, "ListPolicies", `{"MaxResults":10}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
}

func TestFMSStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range fmsOperations {
		resp := fmsRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
