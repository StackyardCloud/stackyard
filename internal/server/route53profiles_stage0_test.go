package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newRoute53ProfilesTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// r53pCall sends a route53profiles-signed REST call and decodes the reply.
func r53pCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "route53profiles")
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
	if want >= 400 && resp.Header.Get("X-Amzn-ErrorType") == "" {
		t.Fatalf("%s %s: error without X-Amzn-ErrorType", method, path)
	}
	return out
}

// r53pErrCode returns the X-Amzn-ErrorType of a call expected to fail.
func r53pErrCode(t *testing.T, ts *httptest.Server, method, path, body string) string {
	t.Helper()
	resp := signedRequestWithService(t, method, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "route53profiles")
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode < 400 {
		t.Fatalf("%s %s: status %d, want an error", method, path, resp.StatusCode)
	}
	return resp.Header.Get("X-Amzn-ErrorType")
}

func TestRoute53ProfilesStage0Catalog(t *testing.T) {
	if len(route53profilesOperations) != 17 || len(route53profilesOperationByName) != 17 {
		t.Fatalf("expected 17 unique operations, got %d/%d", len(route53profilesOperations), len(route53profilesOperationByName))
	}
}

func TestRoute53ProfilesStage0Routing(t *testing.T) {
	_, ts := newRoute53ProfilesTestServer(t)
	r53pCall(t, ts, http.StatusNotFound, http.MethodPut, "/profile/rp-1", "")

	// Route 53 (signing name route53, /2013-04-01/) keeps its traffic.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/2013-04-01/hostedzone", nil, nil, "route53")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "ListHostedZonesResponse") {
		t.Fatalf("route53 ListHostedZones: %d %s", resp.StatusCode, body)
	}
	// A Profiles path signed for another service isn't claimed.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/profiles", nil, nil, "s3")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "ProfileSummaries") {
		t.Fatalf("an s3-signed request reached Route 53 Profiles: %s", body)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/profile", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/route53profiles/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
