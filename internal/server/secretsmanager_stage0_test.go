package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func secretsManagerRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "secretsmanager." + action,
		},
		"secretsmanager",
	)
}

func TestSecretsManagerStage0OperationCoverage(t *testing.T) {
	if len(secretsManagerOperations) != 23 {
		t.Fatalf("expected 23 Secrets Manager operations from docs, got %d", len(secretsManagerOperations))
	}
	if len(secretsManagerOperationByName) != len(secretsManagerOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"CreateSecret",
		"GetSecretValue",
		"PutSecretValue",
		"ListSecrets",
		"DeleteSecret",
		"RotateSecret",
		"ValidateResourcePolicy",
	}
	for _, name := range required {
		if _, ok := secretsManagerOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestSecretsManagerCatalogKnownActionIsHandled(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := secretsManagerRequest(t, ts, "ListSecrets", []byte(`{"MaxResults":10}`))
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected implemented action to avoid %d", http.StatusNotImplemented)
	}
}

func TestSecretsManagerStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := secretsManagerRequest(t, ts, "TotallyUnknownAction", []byte(`{}`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}
