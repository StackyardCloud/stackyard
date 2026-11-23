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

func newRoute53DomainsTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func r53dCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": r53dTargetPrefix + op,
	}, "route53domains")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func r53dOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := r53dCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func r53dErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := r53dCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestRoute53DomainsStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.Route53DomainsModel()
	if len(route53domainsOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(route53domainsOperations), len(model.Operations))
	}
	for _, op := range route53domainsOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if r53dHandlers[op.Name] == nil {
			t.Errorf("%s has no handler", op.Name)
		}
	}
}

func TestRoute53DomainsStage0Routing(t *testing.T) {
	_, ts := newRoute53DomainsTestServer(t)
	if out := r53dOK(t, ts, "ListDomains", `{}`); out["Domains"] == nil {
		t.Fatalf("ListDomains: %v", out)
	}
	// Signed for another service: refused, not served.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": r53dTargetPrefix + "ListDomains",
	}, "route53")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d %s", resp.StatusCode, body)
	}
	if code := r53dErrCode(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	// Model validation: unknown enum value, missing required member.
	if code := r53dErrCode(t, ts, "GetDomainDetail", `{}`); code != "InvalidInput" {
		t.Fatalf("missing DomainName: %s", code)
	}
	// Chunked bodies of unknown length are refused before the body is read.
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Amz-Target", r53dTargetPrefix+"ListDomains")
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
