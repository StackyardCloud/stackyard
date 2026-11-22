package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newGlacierTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// glcCall sends a glacier-signed request and returns status, headers and body.
func glcCall(t *testing.T, ts *httptest.Server, method, path string, body []byte, headers map[string]string) (int, http.Header, []byte) {
	t.Helper()
	h := map[string]string{"x-amz-glacier-version": "2012-06-01"}
	for k, v := range headers {
		h[k] = v
	}
	resp := signedRequestWithService(t, method, ts.URL+path, body, h, "glacier")
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, raw
}

func glcWant(t *testing.T, ts *httptest.Server, want int, method, path string, body []byte, headers map[string]string) (http.Header, map[string]any) {
	t.Helper()
	status, h, raw := glcCall(t, ts, method, path, body, headers)
	if status != want {
		t.Fatalf("%s %s: %d (want %d) %s", method, path, status, want, raw)
	}
	out := map[string]any{}
	if len(raw) > 0 && strings.HasPrefix(h.Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("decode %s: %v", raw, err)
		}
	}
	return h, out
}

func glcErrorCode(t *testing.T, ts *httptest.Server, method, path string, body []byte, headers map[string]string) string {
	t.Helper()
	status, h, raw := glcCall(t, ts, method, path, body, headers)
	if status < 400 {
		t.Fatalf("%s %s: %d, want an error (%s)", method, path, status, raw)
	}
	var e struct{ Code string }
	if err := json.Unmarshal(raw, &e); err != nil || e.Code != h.Get("X-Amzn-ErrorType") {
		t.Fatalf("%s %s: bad error body %s", method, path, raw)
	}
	return e.Code
}

func TestGlacierStage0Catalog(t *testing.T) {
	if len(glacierOperations) != 33 || len(glacierOperationByName) != 33 {
		t.Fatalf("operations: %d listed, %d unique", len(glacierOperations), len(glacierOperationByName))
	}
	for _, name := range []string{"UploadArchive", "InitiateJob", "GetJobOutput", "CompleteMultipartUpload", "CompleteVaultLock"} {
		if _, ok := glacierOperationByName[name]; !ok {
			t.Fatalf("missing %s", name)
		}
	}
}

func TestGlacierStage0Routing(t *testing.T) {
	_, ts := newGlacierTestServer(t)
	if code := glcErrorCode(t, ts, http.MethodPatch, "/-/vaults/v", nil, nil); code != "UnknownOperationException" {
		t.Fatalf("unknown route: %s", code)
	}
	if code := glcErrorCode(t, ts, http.MethodGet, "/999999999999/vaults", nil, nil); code != "AccessDeniedException" {
		t.Fatalf("other account: %s", code)
	}
	if code := glcErrorCode(t, ts, http.MethodGet, "/-/vaults?limit=0", nil, nil); code != "InvalidParameterValueException" {
		t.Fatalf("bad limit: %s", code)
	}
	// An S3-signed request for the same path shape is not Glacier's.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/-/vaults", nil, nil, "s3")
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(raw), "VaultList") || resp.Header.Get("X-Amzn-ErrorType") != "" {
		t.Fatalf("s3-signed request reached Glacier: %d %s", resp.StatusCode, raw)
	}
	// A chunked body (no declared length) is refused before it is read.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/-/vaults/v/archives", io.MultiReader(strings.NewReader("x")))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/glacier/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}

func TestGlacierStage0TreeHash(t *testing.T) {
	if got := glcTreeHash(nil); got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("empty: %s", got)
	}
	// Expected values from botocore.utils.calculate_tree_hash (3 and 5 MiB,
	// i.e. an odd chunk carried up one and two levels).
	for size, want := range map[int]string{
		3 << 20: "2e7d51c0ffe06ce95fe74beed9a4ab35d18837f4ab2a9f4f066a60359eb999a1",
		5 << 20: "1eccd0ca741241ce114bde6d24a8de044ccee139ef6651bec9006d2f558751ef",
	} {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 251)
		}
		if got := glcTreeHash(data); got != want {
			t.Fatalf("%d bytes: %s, want %s", size, got, want)
		}
	}
}
