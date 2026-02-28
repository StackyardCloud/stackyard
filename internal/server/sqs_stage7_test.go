package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSQSStage7SigV4ServiceAndRegion(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := []byte("Action=ListQueues")
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	resp := signRequestWithHeaders(t, http.MethodPost, ts.URL+"/", body, headers, "s3", testRegion, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong service, got %d", resp.StatusCode)
	}

	resp = signRequestWithHeaders(t, http.MethodPost, ts.URL+"/", body, headers, "sqs", "us-west-2", "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong region, got %d", resp.StatusCode)
	}

	resp = signRequestWithHeaders(t, http.MethodPost, ts.URL+"/", body, headers, "sqs", testRegion, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for correct sigv4 scope, got %d", resp.StatusCode)
	}
}
