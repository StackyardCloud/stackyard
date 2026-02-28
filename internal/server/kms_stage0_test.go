package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func kmsRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "TrentService." + action,
		},
		"kms",
	)
}

func TestKMSStage0OperationCoverage(t *testing.T) {
	if len(kmsOperations) != 53 {
		t.Fatalf("expected 53 KMS operations from docs, got %d", len(kmsOperations))
	}
	if len(kmsOperationByName) != len(kmsOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"CreateKey",
		"DescribeKey",
		"Encrypt",
		"Decrypt",
		"ListKeys",
		"ScheduleKeyDeletion",
		"Sign",
		"Verify",
		"RotateKeyOnDemand",
	}
	for _, name := range required {
		if _, ok := kmsOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestKMSKnownActionIsImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := kmsRequest(t, ts, "CreateKey", []byte(`{"Description":"kms-stage-check","KeyUsage":"ENCRYPT_DECRYPT"}`))
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected CreateKey to be implemented, got %d", resp.StatusCode)
	}
}

func TestKMSStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := kmsRequest(t, ts, "TotallyUnknownAction", []byte(`{}`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}
