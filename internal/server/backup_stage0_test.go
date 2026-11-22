package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newBackupTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// backupCall sends a backup-signed REST call and decodes the JSON reply.
func backupCall(t *testing.T, ts *httptest.Server, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != nil {
		raw, _ = json.Marshal(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "backup")
	data := mustBody(t, resp)
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("%s %s: bad JSON %q", method, path, data)
		}
	}
	if resp.StatusCode >= 400 && out["__type"] != resp.Header.Get("X-Amzn-ErrorType") {
		t.Fatalf("%s %s: error type header %q != body %v", method, path, resp.Header.Get("X-Amzn-ErrorType"), out)
	}
	return resp.StatusCode, out
}

// backupWant asserts a status and returns the body.
func backupWant(t *testing.T, ts *httptest.Server, status int, method, path string, body any) map[string]any {
	t.Helper()
	got, out := backupCall(t, ts, method, path, body)
	if got != status {
		t.Fatalf("%s %s: status %d (want %d) %v", method, path, got, status, out)
	}
	return out
}

// backupFail asserts a 400 with the given AWS Backup error code.
func backupFail(t *testing.T, ts *httptest.Server, code, method, path string, body any) {
	t.Helper()
	out := backupWant(t, ts, http.StatusBadRequest, method, path, body)
	if out["__type"] != code {
		t.Fatalf("%s %s: error %v (want %s)", method, path, out["__type"], code)
	}
}

func TestBackupStage0CatalogCoverage(t *testing.T) {
	if len(backupOperations) != 115 || len(backupOperationByName) != 115 {
		t.Fatalf("expected 115 unique operations, got %d/%d", len(backupOperations), len(backupOperationByName))
	}
	for _, op := range backupOperations {
		if backupHandlers[op.Name] == nil {
			t.Errorf("%s has no handler", op.Name)
		}
		// Path variables are input member names, so they start upper-case.
		for _, seg := range strings.Split(op.URI, "{")[1:] {
			if seg[0] < 'A' || seg[0] > 'Z' {
				t.Errorf("%s template variable is not a member name: %s", op.Name, op.URI)
			}
		}
	}
}

func TestBackupStage0RoutesAreUnambiguous(t *testing.T) {
	for _, op := range backupOperations {
		path := op.URI
		for strings.Contains(path, "{") {
			start, end := strings.Index(path, "{"), strings.Index(path, "}")
			path = path[:start] + "x" + path[end+1:]
		}
		got, _ := matchBackupRoute(op.Method, path)
		if got == nil {
			t.Errorf("%s %s matched nothing", op.Method, path)
			continue
		}
		if got.Name != op.Name && !strings.Contains(op.URI, "{") {
			t.Errorf("%s %s matched %s, want %s", op.Method, path, got.Name, op.Name)
		}
	}
	// ARNs travel percent-encoded in a single path segment.
	op, params := matchBackupRoute(http.MethodGet, "/tags/"+url.PathEscape("arn:aws:backup:us-east-1:123456789012:backup-vault:v")+"/")
	if op == nil || op.Name != "ListTags" || params["ResourceArn"] != "arn:aws:backup:us-east-1:123456789012:backup-vault:v" {
		t.Fatalf("ListTags route: %v %v", op, params)
	}
}

func TestBackupStage0OnlyClaimsBackupSignedRequests(t *testing.T) {
	_, ts := newBackupTestServer(t)
	// A DSQL- or S3-signed /tags/ call must not reach AWS Backup.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/backup-vaults/", nil, nil, "s3")
	body := string(mustBody(t, resp))
	if strings.Contains(body, "BackupVaultList") {
		t.Fatalf("s3-signed request reached AWS Backup: %s", body)
	}
	// Unsigned requests are rejected by SigV4 when sent to a backup host.
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/backup-vaults/", nil)
	req.Host = "backup.us-east-1.amazonaws.com"
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusForbidden {
		t.Fatalf("unsigned request: %d", r.StatusCode)
	}
	// An unknown backup route is a clean error, not an S3 fallthrough.
	got, out := backupCall(t, ts, http.MethodGet, "/no-such-route", nil)
	if got != http.StatusNotFound || out["__type"] != "UnknownOperationException" {
		t.Fatalf("unknown route: %d %v", got, out)
	}
}
