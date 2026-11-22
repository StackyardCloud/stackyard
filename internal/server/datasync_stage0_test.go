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

func newDataSyncTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// dsyCall sends a datasync-signed JSON 1.1 request and decodes the reply.
func dsyCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"X-Amz-Target": dsyTargetPrefix + op, "Content-Type": "application/x-amz-json-1.1",
	}, "datasync")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %s: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func dsyOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := dsyCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", op, body, status, out)
	}
	return out
}

// dsyFail expects a 400 and returns its error message.
func dsyFail(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := dsyCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s %s: %d (want 400) %v", op, body, status, out)
	}
	msg, _ := out["message"].(string)
	return msg
}

func TestDataSyncStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.DataSyncModel()
	if len(datasyncOperations) != len(model.Operations) || len(datasyncOperations) != 53 {
		t.Fatalf("operations: %d listed, %d in model", len(datasyncOperations), len(model.Operations))
	}
	for _, op := range datasyncOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Fatalf("%s is not in the model", op.Name)
		}
		if dsyHandlers[op.Name] == nil {
			t.Fatalf("%s has no handler", op.Name)
		}
	}
}

func TestDataSyncStage0Routing(t *testing.T) {
	_, ts := newDataSyncTestServer(t)
	status, out := dsyCall(t, ts, "NoSuchOperation", `{}`)
	if status != http.StatusBadRequest || out["__type"] != "UnknownOperationException" {
		t.Fatalf("unknown operation: %d %v", status, out)
	}
	if msg := dsyFail(t, ts, "ListAgents", `not json`); !strings.Contains(msg, "JSON") {
		t.Fatalf("bad JSON: %s", msg)
	}
	if msg := dsyFail(t, ts, "DescribeAgent", `{}`); !strings.Contains(msg, "AgentArn is required") {
		t.Fatalf("missing required member: %s", msg)
	}
	// Signed for another service: refused, not served.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"X-Amz-Target": dsyTargetPrefix + "ListAgents", "Content-Type": "application/x-amz-json-1.1",
	}, "s3")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d", resp.StatusCode)
	}
	resp.Body.Close()
	// Another JSON 1.1 service still routes past the DataSync hook.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"X-Amz-Target": "AWSGlue.GetDatabases", "Content-Type": "application/x-amz-json-1.1",
	}, "glue")
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestDataSyncStage0RefusesBodiesWithoutDeclaredLength(t *testing.T) {
	_, ts := newDataSyncTestServer(t)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Amz-Target", dsyTargetPrefix+"ListAgents")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}

func TestDataSyncStage0PatternMatch(t *testing.T) {
	for _, c := range []struct {
		pattern, path string
		want          bool
	}{
		{"/skip", "/skip/a.txt", true}, {"/skip", "/skipper.txt", false}, {"*.log", "/dir/c.log", true},
		{"*.log", "/dir/c.txt", false}, {"/dir/*", "/dir/x/y", true}, {"/a*b*c", "/aXbYc", true}, {"/a*b*c", "/aXbY", false},
	} {
		if got := dsyPatternMatch(c.pattern, c.path); got != c.want {
			t.Errorf("dsyPatternMatch(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestDataSyncStage0PatternMatchIsLinear(t *testing.T) {
	// A pathological pattern for backtracking matchers finishes at once.
	pattern := strings.Repeat("*a", 500) + "*b"
	path := "/" + strings.Repeat("a", 1000)
	if dsyPatternMatch(pattern, path) {
		t.Fatal("pattern ending in b matched a path without b")
	}
	if !dsyPatternMatch(pattern, path+"b") {
		t.Fatal("pattern should match the path ending in b")
	}
}
