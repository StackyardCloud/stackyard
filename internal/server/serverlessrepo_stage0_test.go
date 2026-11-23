package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newServerlessRepoTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// sarCall sends a serverlessrepo-signed REST call and decodes the reply.
func sarCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "serverlessrepo")
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("%s %s: %d (want %d) %s", method, path, resp.StatusCode, want, data)
	}
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
	}
	if want >= 400 && (resp.Header.Get("X-Amzn-ErrorType") == "" || out["message"] == nil) {
		t.Fatalf("%s %s: error without X-Amzn-ErrorType or message: %s", method, path, data)
	}
	return out
}

// sarErr returns the error type of a call expected to fail.
func sarErr(t *testing.T, ts *httptest.Server, method, path, body string) string {
	t.Helper()
	resp := signedRequestWithService(t, method, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "serverlessrepo")
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode < 400 {
		t.Fatalf("%s %s: status %d, want an error", method, path, resp.StatusCode)
	}
	return resp.Header.Get("X-Amzn-ErrorType")
}

func TestServerlessRepoStage0Catalog(t *testing.T) {
	if len(serverlessrepoOperations) != 14 || len(serverlessrepoOperationByName) != 14 {
		t.Fatalf("expected 14 unique operations, got %d/%d", len(serverlessrepoOperations), len(serverlessrepoOperationByName))
	}
}

func TestServerlessRepoStage0Routing(t *testing.T) {
	_, ts := newServerlessRepoTestServer(t)
	sarCall(t, ts, http.StatusNotFound, http.MethodPut, "/applications", "")

	// A request signed for another service isn't claimed: create an app, then
	// list /applications signed for s3 and check it isn't in the answer.
	sarCall(t, ts, http.StatusCreated, http.MethodPost, "/applications", `{"name":"routed","author":"a","description":"d"}`)
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/applications", nil, nil, "s3")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "routed") {
		t.Fatalf("an s3-signed request reached serverlessrepo: %s", body)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/applications", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/serverlessrepo/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
