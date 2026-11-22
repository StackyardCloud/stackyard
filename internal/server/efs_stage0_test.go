package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newEFSTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func efsRequest(t *testing.T, ts *httptest.Server, method, path, body string) *http.Response {
	t.Helper()
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	return signedRequestWithService(t, method, ts.URL+"/2015-02-01/"+path, raw, map[string]string{"Content-Type": "application/json"}, "elasticfilesystem")
}

// efsCall sends a request, checks the status and decodes the JSON reply.
func efsCall(t *testing.T, ts *httptest.Server, method, path, body string, want int) map[string]any {
	t.Helper()
	resp := efsRequest(t, ts, method, path, body)
	assertStatus(t, resp, want)
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, path, raw, err)
		}
	}
	return out
}

// efsCallError checks an error reply's status and ErrorCode.
func efsCallError(t *testing.T, ts *httptest.Server, method, path, body string, want int, code string) map[string]any {
	t.Helper()
	resp := efsRequest(t, ts, method, path, body)
	if got := resp.Header.Get("X-Amzn-ErrorType"); got != code {
		t.Errorf("%s %s: X-Amzn-ErrorType = %q, want %q", method, path, got, code)
	}
	out := efsCallBody(t, resp, want)
	if out["ErrorCode"] != code {
		t.Fatalf("%s %s: ErrorCode = %v, want %s (%v)", method, path, out["ErrorCode"], code, out)
	}
	return out
}

func efsCallBody(t *testing.T, resp *http.Response, want int) map[string]any {
	t.Helper()
	assertStatus(t, resp, want)
	out := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func efsCreateFS(t *testing.T, ts *httptest.Server, body string) string {
	t.Helper()
	out := efsCall(t, ts, http.MethodPost, "file-systems", body, http.StatusCreated)
	id, _ := out["FileSystemId"].(string)
	if !strings.HasPrefix(id, "fs-") {
		t.Fatalf("CreateFileSystem: no FileSystemId in %v", out)
	}
	return id
}

func TestEFSStage0Operations(t *testing.T) {
	if len(efsOperations) != 31 {
		t.Fatalf("efsOperations = %d, want 31", len(efsOperations))
	}
	if len(efsOperationByName) != len(efsOperations) {
		t.Fatalf("operation names are not unique")
	}
	for _, name := range []string{"CreateFileSystem", "CreateMountTarget", "CreateAccessPoint", "CreateReplicationConfiguration", "DescribeTags", "UpdateFileSystemProtection"} {
		if _, ok := efsOperationByName[name]; !ok {
			t.Fatalf("missing operation %s", name)
		}
	}
}

func TestEFSStage0Routing(t *testing.T) {
	_, ts := newEFSTestServer(t)
	efsCallError(t, ts, http.MethodGet, "no-such-thing", "", http.StatusNotImplemented, "NotImplementedException")

	// Signed for another service: the EFS router doesn't claim it.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/2015-02-01/file-systems", nil, nil, "s3")
	if resp.StatusCode == http.StatusOK && strings.Contains(string(mustBody(t, resp)), "FileSystems") {
		t.Fatalf("an s3-signed request reached EFS")
	}

	// Unsigned requests are rejected by SigV4.
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/2015-02-01/file-systems", nil)
	req.Host = "elasticfilesystem.us-east-1.amazonaws.com"
	unsigned, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if unsigned.StatusCode != http.StatusForbidden {
		t.Fatalf("unsigned request: %d, want 403", unsigned.StatusCode)
	}
	unsigned.Body.Close()

	efsCallError(t, ts, http.MethodPost, "file-systems", "{bad json", http.StatusBadRequest, "BadRequest")
	efsCallError(t, ts, http.MethodGet, "file-systems?MaxItems=0", "", http.StatusBadRequest, "BadRequest")
	efsCallError(t, ts, http.MethodGet, "file-systems?MaxItems=99999999999", "", http.StatusBadRequest, "BadRequest")
}
