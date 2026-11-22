package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newCodeStarNotificationsTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// csnCall sends a signed CodeStar Notifications call and decodes the reply.
func csnCall(t *testing.T, ts *httptest.Server, want int, path, body string) map[string]any {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "codestar-notifications")
	raw := mustBody(t, resp)
	if resp.StatusCode != want {
		t.Fatalf("POST %s: %d (want %d) %s", path, resp.StatusCode, want, raw)
	}
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("decode %s: %v", raw, err)
		}
	}
	return out
}

func csnErrorType(t *testing.T, ts *httptest.Server, path, body string) string {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "codestar-notifications")
	_ = mustBody(t, resp)
	if resp.StatusCode < 400 {
		t.Fatalf("POST %s: %d, want an error", path, resp.StatusCode)
	}
	return resp.Header.Get("X-Amzn-ErrorType")
}

func TestCodeStarNotificationsStage0Catalog(t *testing.T) {
	if len(codestarnotificationsOperations) != 13 || len(codestarnotificationsOperationByPath) != 13 {
		t.Fatalf("operations: %d (by path %d), want 13", len(codestarnotificationsOperations), len(codestarnotificationsOperationByPath))
	}
	for _, name := range []string{"CreateNotificationRule", "Subscribe", "UntagResource", "ListEventTypes"} {
		found := false
		for _, op := range codestarnotificationsOperations {
			found = found || op.Name == name
		}
		if !found {
			t.Fatalf("missing operation %s", name)
		}
	}
	for _, e := range csnEventTypes {
		if !strings.HasPrefix(e.EventTypeId, strings.ToLower(e.ServiceName)+"-") {
			t.Fatalf("event type %s doesn't match service %s", e.EventTypeId, e.ServiceName)
		}
	}
}

func TestCodeStarNotificationsStage0Routing(t *testing.T) {
	_, ts := newCodeStarNotificationsTestServer(t)
	// Signed for another service: not claimed (S3 answers instead).
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/listTargets", []byte(`{}`), nil, "s3")
	_ = mustBody(t, resp)
	if resp.Header.Get("X-Amzn-ErrorType") != "" {
		t.Fatalf("s3-signed request reached CodeStar Notifications")
	}
	// Wrong method.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/listTargets", nil, nil, "codestar-notifications")
	_ = mustBody(t, resp)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("GET /listTargets: %d", resp.StatusCode)
	}
	// Unknown members are rejected, so callers can't set internal fields.
	if code := csnErrorType(t, ts, "/listTargets", `{"bogus":1}`); code != "ValidationException" {
		t.Fatalf("unknown member: %s", code)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/listTargets", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/codestar-notifications/aws4_request, SignedHeaders=host, Signature=00")
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(r2.Body)
	r2.Body.Close()
	if r2.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Content-Length") {
		t.Fatalf("chunked body: %d %s", r2.StatusCode, body)
	}
}
