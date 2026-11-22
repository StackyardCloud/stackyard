package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newPipesTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// pipesCall sends a pipes-signed REST call and decodes the JSON reply.
func pipesCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "pipes")
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

func TestPipesStage0Catalog(t *testing.T) {
	if len(pipesOperations) != 10 || len(pipesOperationByName) != 10 {
		t.Fatalf("expected 10 unique operations, got %d/%d", len(pipesOperations), len(pipesOperationByName))
	}
	for _, name := range []string{"CreatePipe", "DescribePipe", "UpdatePipe", "DeletePipe", "ListPipes", "StartPipe", "StopPipe", "TagResource", "UntagResource", "ListTagsForResource"} {
		if _, ok := pipesOperationByName[name]; !ok {
			t.Fatalf("missing operation %s", name)
		}
	}
}

func TestPipesStage0Routing(t *testing.T) {
	_, ts := newPipesTestServer(t)
	// Unknown pipes route.
	pipesCall(t, ts, http.StatusNotFound, http.MethodPatch, "/v1/pipes/p", "")
	// Signed for another service: not claimed by Pipes (S3 answers instead).
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/v1/pipes", nil, nil, "s3")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), `"Pipes"`) {
		t.Fatalf("an s3-signed request reached Pipes: %s", body)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/pipes/p", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/pipes/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
