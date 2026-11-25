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

func newBudTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// budCallAPI sends a signed awsJson request for api and decodes the reply.
func budCallAPI(t *testing.T, ts *httptest.Server, api *budJSONAPI, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": api.contentType, "X-Amz-Target": api.prefix + op,
	}, api.signing)
	raw := mustBody(t, resp)
	if ct := resp.Header.Get("Content-Type"); ct != api.contentType {
		t.Fatalf("%s: content type %q", op, ct)
	}
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func budOKAPI(t *testing.T, ts *httptest.Server, api *budJSONAPI, op, body string) map[string]any {
	t.Helper()
	status, out := budCallAPI(t, ts, api, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

// budErrAPI expects a 4xx and returns the error code.
func budErrAPI(t *testing.T, ts *httptest.Server, api *budJSONAPI, op, body string, wantStatus int) string {
	t.Helper()
	status, out := budCallAPI(t, ts, api, op, body)
	if status != wantStatus {
		t.Fatalf("%s: want %d, got %d %v", op, wantStatus, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func budOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	return budOKAPI(t, ts, budAPI, op, body)
}

func budErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	return budErrAPI(t, ts, budAPI, op, body, http.StatusBadRequest)
}

func TestBudgetsStage0CatalogsMatchModels(t *testing.T) {
	for _, c := range []struct {
		name  string
		ops   []string
		model *awsmodels.Model
		api   *budJSONAPI
	}{
		{"budgets", budOpNames(), awsmodels.BudgetsModel(), budAPI},
		{"cost-optimization-hub", cohOpNames(), awsmodels.CostOptimizationHubModel(), cohAPI},
		{"bcm-data-exports", bcmdeOpNames(), awsmodels.BCMDataExportsModel(), bcmdeAPI},
	} {
		if len(c.ops) != len(c.model.Operations) || len(c.api.handlers) != len(c.ops) {
			t.Fatalf("%s: catalog %d, model %d, handlers %d", c.name, len(c.ops), len(c.model.Operations), len(c.api.handlers))
		}
		for _, op := range c.ops {
			if _, ok := c.model.Operations[op]; !ok {
				t.Errorf("%s: %s is not in the model", c.name, op)
			}
			if c.api.handlers[op] == nil {
				t.Errorf("%s: %s has no handler", c.name, op)
			}
		}
	}
}

func budOpNames() []string {
	out := make([]string, 0, len(budOperations))
	for _, op := range budOperations {
		out = append(out, op.Name)
	}
	return out
}

func cohOpNames() []string {
	out := make([]string, 0, len(cohOperations))
	for _, op := range cohOperations {
		out = append(out, op.Name)
	}
	return out
}

func bcmdeOpNames() []string {
	out := make([]string, 0, len(bcmdeOperations))
	for _, op := range bcmdeOperations {
		out = append(out, op.Name)
	}
	return out
}

func TestBudgetsStage0Routing(t *testing.T) {
	_, ts := newBudTestServer(t)
	budOK(t, ts, "DescribeBudgets", `{"AccountId":"123456789012"}`)
	budOKAPI(t, ts, cohAPI, "GetPreferences", `{}`)
	budOKAPI(t, ts, bcmdeAPI, "ListTables", `{}`)

	// Signed for another service: never served as Budgets.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{"AccountId":"123456789012"}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": budTargetPrefix + "DescribeBudgets",
	}, "s3")
	if body := mustBody(t, resp); resp.StatusCode == http.StatusOK {
		t.Fatalf("s3-signed request was served: %s", body)
	}

	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", budTargetPrefix+"DescribeBudgets")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r.StatusCode)
	}

	// Oversized bodies are refused.
	big := `{"AccountId":"123456789012","Pad":"` + strings.Repeat("x", budMaxBody) + `"}`
	if code := budErrCode(t, ts, "DescribeBudgets", big); code != "InvalidParameterException" {
		t.Fatalf("oversized body: %s", code)
	}

	// Unknown members are refused by the model.
	if code := budErrCode(t, ts, "DescribeBudgets", `{"AccountId":"123456789012","Nope":1}`); code != "InvalidParameterException" {
		t.Fatalf("unknown member: %s", code)
	}

	// Cost Explorer keeps its own traffic through the cost-management router.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSInsightsIndexService.GetAnomalyMonitors",
	}, "ce")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK {
		t.Fatalf("ce GetAnomalyMonitors: %d %s", resp.StatusCode, body)
	}
}
