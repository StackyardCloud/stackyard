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

func newServiceCatalogTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func svcatCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": svcatTargetPrefix + op,
	}, "servicecatalog")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func svcatOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := svcatCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func svcatErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := svcatCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestServiceCatalogStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.ServiceCatalogModel()
	if len(serviceCatalogOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(serviceCatalogOperations), len(model.Operations))
	}
	for _, op := range serviceCatalogOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if svcatHandlers[op.Name] == nil {
			t.Errorf("%s has no handler", op.Name)
		}
	}
}

func TestServiceCatalogStage0Routing(t *testing.T) {
	_, ts := newServiceCatalogTestServer(t)
	if out := svcatOK(t, ts, "ListPortfolios", `{}`); out["PortfolioDetails"] == nil {
		t.Fatalf("ListPortfolios: %v", out)
	}
	// Signed for another service: refused, not served.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": svcatTargetPrefix + "ListPortfolios",
	}, "cloudformation")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d %s", resp.StatusCode, body)
	}
	if code := svcatErrCode(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	if code := svcatErrCode(t, ts, "DescribePortfolio", `{}`); code != "InvalidParametersException" {
		t.Fatalf("missing Id: %s", code)
	}
	// Callers can't reach the store's own fields: unknown members are refused.
	if code := svcatErrCode(t, ts, "ListPortfolios", `{"portfolios":{}}`); code != "InvalidParametersException" {
		t.Fatalf("unknown member: %s", code)
	}
	// Chunked bodies of unknown length are refused before the body is read.
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Amz-Target", svcatTargetPrefix+"ListPortfolios")
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
	// CloudFormation (query protocol) still answers on its own.
	cf := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte("Action=ListStacks&Version=2010-05-15"),
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "cloudformation")
	if body := mustBody(t, cf); cf.StatusCode != http.StatusOK || !strings.Contains(string(body), "ListStacksResponse") {
		t.Fatalf("cloudformation: %d %s", cf.StatusCode, body)
	}
}
