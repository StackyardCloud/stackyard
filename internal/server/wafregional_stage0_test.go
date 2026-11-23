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

func newWAFRegionalTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func wrCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": wrTargetPrefix + op,
	}, "waf-regional")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func wrOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := wrCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func wrErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := wrCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

// wrToken returns a fresh change token.
func wrToken(t *testing.T, ts *httptest.Server) string {
	t.Helper()
	return wrOK(t, ts, "GetChangeToken", `{}`)["ChangeToken"].(string)
}

func TestWAFRegionalStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.WAFRegionalModel()
	if len(wafRegionalOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(wafRegionalOperations), len(model.Operations))
	}
	for _, op := range wafRegionalOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if wafRegionalHandlers[op.Name] == nil && op.Name != "CreateWebACLMigrationStack" {
			t.Errorf("%s has no handler", op.Name)
		}
	}
}

func TestWAFRegionalStage0Routing(t *testing.T) {
	_, ts := newWAFRegionalTestServer(t)
	if out := wrOK(t, ts, "ListIPSets", `{}`); out["IPSets"] == nil {
		t.Fatalf("ListIPSets: %v", out)
	}
	// Signed for another service: refused, not served.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": wrTargetPrefix + "ListIPSets",
	}, "wafv2")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d %s", resp.StatusCode, body)
	}
	if code := wrErrCode(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	// WAFv2 keeps its own traffic.
	v2 := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{"Scope":"REGIONAL"}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSWAF_20190729.ListWebACLs",
	}, "wafv2")
	if body := mustBody(t, v2); v2.StatusCode != http.StatusOK || strings.Contains(string(body), "IPSets") {
		t.Fatalf("WAFv2 ListWebACLs: %d %s", v2.StatusCode, body)
	}
	// Chunked bodies of unknown length are refused.
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Amz-Target", wrTargetPrefix+"ListIPSets")
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
}
