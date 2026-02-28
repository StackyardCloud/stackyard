package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignerStage0OperationCoverage(t *testing.T) {
	if len(signerOperations) != 19 {
		t.Fatalf("expected 19 Signer operations from docs, got %d", len(signerOperations))
	}
	if len(signerOperationByName) != len(signerOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"PutSigningProfile",
		"StartSigningJob",
		"DescribeSigningJob",
		"SignPayload",
		"GetRevocationStatus",
	}
	for _, name := range required {
		if _, ok := signerOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
	if _, ok := signerOperationByName["GetSigningJob"]; ok {
		t.Fatalf("unexpected operation GetSigningJob present in signer catalog")
	}
}

func signerRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{"Content-Type": "application/json"}
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		body,
		headers,
		"signer",
	)
}

func TestSignerStage0KnownRouteIsImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signerRequest(t, ts, http.MethodGet, "/signing-profiles", nil)
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected ListSigningProfiles to be implemented")
	}

	resp = signerRequest(t, ts, http.MethodGet, "/totally-unknown-signer-route", nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected unknown route to return %d, got %d", http.StatusNotImplemented, resp.StatusCode)
	}
}
