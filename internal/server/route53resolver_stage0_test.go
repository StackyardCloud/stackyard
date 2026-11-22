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

func newRoute53ResolverTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func r53rRequest(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "Route53Resolver." + op,
	}, "route53resolver")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %s: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func r53rOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := r53rRequest(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: status %d %v", op, body, status, out)
	}
	return out
}

func r53rFail(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := r53rRequest(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s %s: status %d (want 400) %v", op, body, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestRoute53ResolverStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.Route53ResolverModel()
	if len(route53resolverOperations) != len(model.Operations) || len(route53resolverOperationByName) != len(model.Operations) {
		t.Fatalf("catalog has %d ops, model %d", len(route53resolverOperations), len(model.Operations))
	}
	for _, op := range route53resolverOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Fatalf("%s isn't in the model", op.Name)
		}
		if r53rHandlers[op.Name] == nil {
			t.Fatalf("%s has no handler", op.Name)
		}
	}
}

func TestRoute53ResolverStage0RoutingAndRequestChecks(t *testing.T) {
	_, ts := newRoute53ResolverTestServer(t)
	if code := r53rFail(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	// Missing required members are model-validation errors.
	if code := r53rFail(t, ts, "GetResolverEndpoint", `{}`); code != "InvalidParameterException" {
		t.Fatalf("missing member: %s", code)
	}
	if code := r53rFail(t, ts, "GetFirewallRuleGroup", `{}`); code != "ValidationException" {
		t.Fatalf("missing firewall member: %s", code)
	}
	if code := r53rFail(t, ts, "ListResolverRules", `not json`); code != "SerializationException" {
		t.Fatalf("bad json: %s", code)
	}

	// A request signed for another service isn't accepted.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "Route53Resolver.ListResolverRules",
	}, "route53")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Route 53 itself (REST-XML) still routes to Route 53.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/2013-04-01/hostedzone", nil, nil, "route53")
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ListHostedZonesResponse") {
		t.Fatalf("route53 response: %s", body)
	}

	// Bodies of unknown length are refused before signing.
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Amz-Target", "Route53Resolver.ListResolverRules")
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Body.Close()
	if r2.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r2.StatusCode)
	}
}
