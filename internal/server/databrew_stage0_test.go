package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func newDataBrewTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// dbwCall sends a databrew-signed REST call and decodes the JSON reply.
func dbwCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	code, out := dbwRequest(t, ts, method, path, body)
	if code != want {
		t.Fatalf("%s %s: %d (want %d) %v", method, path, code, want, out)
	}
	return out
}

func dbwRequest(t *testing.T, ts *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "databrew")
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
	}
	if resp.StatusCode >= 400 {
		out["__type"] = resp.Header.Get("X-Amzn-ErrorType")
	}
	return resp.StatusCode, out
}

// dbwErrType returns the error type of a call expected to fail.
func dbwErrType(t *testing.T, ts *httptest.Server, method, path, body string) string {
	t.Helper()
	code, out := dbwRequest(t, ts, method, path, body)
	if code < 400 {
		t.Fatalf("%s %s: %d, want an error", method, path, code)
	}
	return out["__type"].(string)
}

func TestDataBrewStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.DataBrewModel()
	if len(dataBrewOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model has %d", len(dataBrewOperations), len(model.Operations))
	}
	for _, op := range dataBrewOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Fatalf("catalog operation %s isn't in the model", op.Name)
		}
	}
}

func TestDataBrewStage0Routing(t *testing.T) {
	_, ts := newDataBrewTestServer(t)
	dbwCall(t, ts, http.StatusOK, http.MethodGet, "/datasets", "")
	if code := dbwErrType(t, ts, http.MethodGet, "/datasets/a/b/c", ""); code != "UnknownOperationException" {
		t.Fatalf("unknown route: %s", code)
	}
	// An S3-signed request to /jobs is not DataBrew's.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/jobs", nil, nil, "s3")
	resp.Body.Close()
	if resp.Header.Get("X-Amzn-ErrorType") == "UnknownOperationException" {
		t.Fatal("DataBrew claimed an S3-signed request")
	}
	// Bodies of unknown length are refused.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/datasets", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/databrew/aws4_request, SignedHeaders=host, Signature=00")
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r2.StatusCode)
	}
}
