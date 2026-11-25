package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func newConnectTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// cnxDo sends a request signed for service and decodes a JSON reply.
func cnxDo(t *testing.T, ts *httptest.Server, service, method, path, body string, headers map[string]string) (int, map[string]any) {
	t.Helper()
	if headers == nil {
		headers = map[string]string{}
	}
	headers["Content-Type"] = "application/json"
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, service)
	data := mustBody(t, resp)
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, path, data, err)
		}
	}
	return resp.StatusCode, out
}

func cncOK(t *testing.T, ts *httptest.Server, method, path, body string) map[string]any {
	t.Helper()
	status, out := cnxDo(t, ts, "cases", method, path, body, nil)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", method, path, status, out)
	}
	return out
}

func cncFail(t *testing.T, ts *httptest.Server, method, path, body string, want int) string {
	t.Helper()
	status, out := cnxDo(t, ts, "cases", method, path, body, nil)
	if status != want {
		t.Fatalf("%s %s: want %d, got %d %v", method, path, want, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestConnectCasesStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.ConnectCasesModel()
	if len(connectCasesOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(connectCasesOperations), len(model.Operations))
	}
	for _, op := range connectCasesOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if cncHandlers[op.Name] == nil {
			t.Errorf("%s has no handler", op.Name)
		}
		// Every template routes back to its own operation.
		path := strings.NewReplacer("{domainId}", "d1", "{caseId}", "c1", "{fieldId}", "f1", "{layoutId}", "l1",
			"{templateId}", "t1", "{caseRuleId}", "r1", "{relatedItemId}", "i1", "{arn}", "arn%3Aaws%3Acases%3Ax").Replace(op.URI)
		if got, _ := cncRoute(op.Method, path); got != op.Name {
			t.Errorf("%s %s routes to %q, want %s", op.Method, path, got, op.Name)
		}
	}
}

func TestConnectCasesStage0Routing(t *testing.T) {
	_, ts := newConnectTestServer(t)
	cncOK(t, ts, http.MethodPost, "/domains-list", "")
	if code := cncFail(t, ts, http.MethodPost, "/frobnicate", "{}", http.StatusNotFound); code != "UnknownOperationException" {
		t.Fatalf("unknown path: %s", code)
	}
	if code := cncFail(t, ts, http.MethodPost, "/domains", `{"name":`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad json: %s", code)
	}
	if code := cncFail(t, ts, http.MethodPost, "/domains", `{}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("missing name: %s", code)
	}
	if code := cncFail(t, ts, http.MethodPost, "/domains-list?maxResults=500", "", http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("maxResults: %s", code)
	}

	// /tags/{arn} signed for another service is never claimed by Cases.
	arn := "arn%3Aaws%3Acases%3Aus-east-1%3A123456789012%3Adomain%2Fnope"
	if status, _ := cnxDo(t, ts, "cases", http.MethodGet, "/tags/"+arn, "", nil); status != http.StatusNotFound {
		t.Fatalf("cases tags on a missing resource: %d", status)
	}
	if status, out := cnxDo(t, ts, "connect", http.MethodGet, "/tags/"+arn, "", nil); status != http.StatusOK || out["tags"] == nil {
		t.Fatalf("connect should keep /tags: %d %v", status, out)
	}

	// A chunked body of unknown length is refused.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/domains", io.MultiReader(strings.NewReader(`{"name":"x"}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=x/20260101/us-east-1/cases/aws4_request, SignedHeaders=host, Signature=00")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r.StatusCode)
	}
}

// cncFixture creates a domain with a Number field, a layout and a template
// requiring the title.
func cncFixture(t *testing.T, ts *httptest.Server) (domain, field, template string) {
	t.Helper()
	domain = cncOK(t, ts, http.MethodPost, "/domains", `{"name":"support"}`)["domainId"].(string)
	field = cncOK(t, ts, http.MethodPost, "/domains/"+domain+"/fields", `{"name":"priority","type":"Number"}`)["fieldId"].(string)
	layout := cncOK(t, ts, http.MethodPost, "/domains/"+domain+"/layouts",
		fmt.Sprintf(`{"name":"main","content":{"basic":{"topPanel":{"sections":[{"fieldGroup":{"fields":[{"id":"title"},{"id":%q}]}}]}}}}`, field))["layoutId"].(string)
	template = cncOK(t, ts, http.MethodPost, "/domains/"+domain+"/templates",
		fmt.Sprintf(`{"name":"default","layoutConfiguration":{"defaultLayout":%q},"requiredFields":[{"fieldId":"title"}],"tagPropagationConfigurations":[{"resourceType":"Cases","tagMap":{"team":"support"}}]}`, layout))["templateId"].(string)
	return
}

func cncCreateCase(t *testing.T, ts *httptest.Server, domain, template, title string, priority float64) string {
	t.Helper()
	return cncOK(t, ts, http.MethodPost, "/domains/"+domain+"/cases", fmt.Sprintf(
		`{"templateId":%q,"fields":[{"id":"title","value":{"stringValue":%q}},{"id":"status","value":{"stringValue":"open"}}],"performedBy":{"customEntity":"bot"}}`,
		template, title))["caseId"].(string)
}
