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

// acctCallREST sends a signed REST JSON request to path and decodes the reply.
func acctCallREST(t *testing.T, ts *httptest.Server, signing, path, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, signing)
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", path, raw, err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		out["__type"] = resp.Header.Get("X-Amzn-ErrorType")
	}
	return resp.StatusCode, out
}

func acctOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := acctCallREST(t, ts, "account", acctPathOf(op), body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

// acctErr expects wantStatus and returns the error code.
func acctErr(t *testing.T, ts *httptest.Server, op, body string, wantStatus int) string {
	t.Helper()
	status, out := acctCallREST(t, ts, "account", acctPathOf(op), body)
	if status != wantStatus {
		t.Fatalf("%s: want %d, got %d %v", op, wantStatus, status, out)
	}
	return out["__type"].(string)
}

func acctPathOf(op string) string {
	for p, name := range acctPaths {
		if name == op {
			return p
		}
	}
	panic("no path for " + op)
}

func TestAccountStage0CatalogsMatchModels(t *testing.T) {
	names := func(n int, at func(int) string) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = at(i)
		}
		return out
	}
	for _, c := range []struct {
		name  string
		ops   []string
		model *awsmodels.Model
		api   *budJSONAPI
	}{
		{"account", names(len(acctOperations), func(i int) string { return acctOperations[i].Name }), awsmodels.AccountModel(), acctAPI},
		{"freetier", names(len(ftierOperations), func(i int) string { return ftierOperations[i].Name }), awsmodels.FreeTierModel(), ftierAPI},
		{"taxsettings", names(len(taxsOperations), func(i int) string { return taxsOperations[i].Name }), awsmodels.TaxSettingsModel(), taxsAPI},
	} {
		if len(c.ops) != len(c.model.Operations) || len(c.api.handlers) != len(c.ops) {
			t.Fatalf("%s: catalog %d, model %d, handlers %d", c.name, len(c.ops), len(c.model.Operations), len(c.api.handlers))
		}
		for _, op := range c.ops {
			if _, ok := c.model.Operations[op]; !ok || c.api.handlers[op] == nil {
				t.Errorf("%s: %s missing from the model or handlers", c.name, op)
			}
		}
	}
	if len(acctPaths) != len(acctOperations) || acctPaths["/putAlternateContact"] != "PutAlternateContact" {
		t.Fatalf("account paths: %v", acctPaths)
	}
	if taxsPaths["/PutTaxRegistration"] != "PutTaxRegistration" {
		t.Fatalf("tax paths: %v", taxsPaths)
	}
}

func TestAccountStage0Routing(t *testing.T) {
	_, ts := newBudTestServer(t)
	acctOK(t, ts, "GetAccountInformation", `{}`)
	budOKAPI(t, ts, ftierAPI, "GetAccountPlanState", `{}`)
	if status, out := acctCallREST(t, ts, "tax", "/GetTaxInheritance", `{}`); status != http.StatusOK || out["heritageStatus"] != "OptOut" {
		t.Fatalf("GetTaxInheritance: %d %v", status, out)
	}

	// Account paths signed for another service are not served as Account.
	if status, out := acctCallREST(t, ts, "s3", "/getAccountInformation", `{}`); status == http.StatusOK && out["AccountId"] != nil {
		t.Fatalf("s3-signed request served as Account: %v", out)
	}

	// Neighbours keep their traffic: Organizations, Budgets, Cost Explorer
	// (cost-management router) and Billing Conductor.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{"AccountId":"210987654321"}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSOrganizationsV20161128.DescribeAccount",
	}, "organizations")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Account"`) {
		t.Fatalf("organizations DescribeAccount: %d %s", resp.StatusCode, body)
	}
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSOrganizationsV20161128.ListAccounts",
	}, "organizations")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "210987654321") {
		t.Fatalf("organizations ListAccounts: %d %s", resp.StatusCode, body)
	}
	budOK(t, ts, "DescribeBudgets", `{"AccountId":"123456789012"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSInsightsIndexService.GetAnomalyMonitors",
	}, "ce")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK {
		t.Fatalf("ce GetAnomalyMonitors: %d %s", resp.StatusCode, body)
	}
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/list-billing-groups", []byte(`{}`), map[string]string{"Content-Type": "application/json"}, "billingconductor")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "BillingGroups") {
		t.Fatalf("billingconductor ListBillingGroups: %d %s", resp.StatusCode, body)
	}

	// A chunked body of unknown length is refused.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/getAccountInformation", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=x/20260101/us-east-1/account/aws4_request, SignedHeaders=host, Signature=0")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r.StatusCode)
	}

	// Oversized bodies and unknown members are refused.
	big := `{"Pad":"` + strings.Repeat("x", budMaxBody) + `"}`
	if code := acctErr(t, ts, "GetAccountInformation", big, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("oversized body: %s", code)
	}
	if code := acctErr(t, ts, "GetAccountInformation", `{"Nope":1}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown member: %s", code)
	}
}
