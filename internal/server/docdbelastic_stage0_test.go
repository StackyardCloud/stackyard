package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newDocDBElasticTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// dbeCall sends a docdb-elastic-signed REST call and decodes the JSON reply.
func dbeCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "docdb-elastic")
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

func dbeEsc(arn string) string { return url.PathEscape(arn) }

func TestDocDBElasticStage0Catalog(t *testing.T) {
	if len(docdbelasticOperations) != 19 || len(docdbelasticOperationByName) != 19 {
		t.Fatalf("expected 19 unique operations, got %d/%d", len(docdbelasticOperations), len(docdbelasticOperationByName))
	}
}

func TestDocDBElasticStage0Routing(t *testing.T) {
	_, ts := newDocDBElasticTestServer(t)
	dbeCall(t, ts, http.StatusOK, http.MethodGet, "/clusters", "")
	dbeCall(t, ts, http.StatusNotFound, http.MethodPatch, "/cluster/x", "")
	// The instance-based docdb CLI hint must not match docdb-elastic traffic.
	if hasDocDBRequestHint(&http.Request{Header: http.Header{"User-Agent": {"aws-cli/2 md/command#docdb-elastic.list-clusters"}}, URL: &url.URL{Path: "/clusters"}}) {
		t.Fatal("docdb-elastic CLI traffic matched the docdb hint")
	}
	// Signed for another service, /clusters isn't docdb-elastic's (EKS answers it).
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/clusters", nil, nil, "eks")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "DocumentDB Elastic") || strings.Contains(string(body), "docdb-elastic") {
		t.Fatalf("an eks-signed request reached docdb-elastic: %s", body)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/cluster", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/docdb-elastic/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
