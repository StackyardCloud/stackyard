package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Stage 0 for the Dashboards / Pricing Calculator / Application Cost
// Profiler batch: catalogs match the CLI models, and routing keeps the
// neighbouring cost-management services' traffic where it was.

func TestBCMBatchStage0CatalogsMatchModels(t *testing.T) {
	names := func(n int, at func(int) string) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = at(i)
		}
		return out
	}
	for _, c := range []struct {
		name     string
		ops      []string
		model    *awsmodels.Model
		handlers map[string]budHandler
	}{
		{"bcm-dashboards", names(len(bcmdOperations), func(i int) string { return bcmdOperations[i].Name }), awsmodels.BCMDashboardsModel(), bcmdAPI.handlers},
		{"bcm-pricing-calculator", names(len(bcmpcOperations), func(i int) string { return bcmpcOperations[i].Name }), awsmodels.BCMPricingCalculatorModel(), bcmpcAPI.handlers},
	} {
		if len(c.ops) != len(c.model.Operations) || len(c.handlers) != len(c.ops) {
			t.Fatalf("%s: catalog %d, model %d, handlers %d", c.name, len(c.ops), len(c.model.Operations), len(c.handlers))
		}
		for _, op := range c.ops {
			if _, ok := c.model.Operations[op]; !ok || c.handlers[op] == nil {
				t.Errorf("%s: %s missing from model or handlers", c.name, op)
			}
		}
	}
	srv, _ := newBudTestServer(t)
	acp := awsmodels.ApplicationCostProfilerModel()
	if len(acpOperations) != len(acp.Operations) {
		t.Fatalf("application-cost-profiler: catalog %d, model %d", len(acpOperations), len(acp.Operations))
	}
	for _, op := range acpOperations {
		if _, ok := acp.Operations[op.Name]; !ok {
			t.Errorf("application-cost-profiler: %s is not in the model", op.Name)
		}
		if _, err := srv.acpDispatch(op.Name, map[string]any{}); err != nil && err.code == "UnknownOperationException" {
			t.Errorf("application-cost-profiler: %s has no handler", op.Name)
		}
	}
}

func TestBCMBatchStage0Routing(t *testing.T) {
	_, ts := newBudTestServer(t)
	budOKAPI(t, ts, bcmdAPI, "ListDashboards", `{}`)
	budOKAPI(t, ts, bcmpcAPI, "GetPreferences", `{}`)
	if status, _ := acpCall(t, ts, http.MethodGet, "/reportDefinition", ""); status != http.StatusOK {
		t.Fatalf("ListReportDefinitions: %d", status)
	}

	// Neighbours keep their traffic.
	budOK(t, ts, "DescribeBudgets", `{"AccountId":"123456789012"}`)
	budOKAPI(t, ts, cohAPI, "GetPreferences", `{}`)
	budOKAPI(t, ts, bcmdeAPI, "ListTables", `{}`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSInsightsIndexService.GetAnomalyMonitors",
	}, "ce")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK {
		t.Fatalf("ce GetAnomalyMonitors: %d %s", resp.StatusCode, body)
	}
	resp = billingConductorRequest(t, ts, http.MethodPost, "/list-billing-groups", `{}`)
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK {
		t.Fatalf("billingconductor ListBillingGroups: %d %s", resp.StatusCode, body)
	}
	// The cost-management router still serves Pricing Calculator actions sent
	// under its own target prefix.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSBillingAndCostManagement.AWSBCMPricingCalculator_GetPreferences",
	}, "bcm-pricing-calculator")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK {
		t.Fatalf("cost-management pricing action: %d %s", resp.StatusCode, body)
	}

	// Signed for another service: never served as Dashboards.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": bcmdContentType, "X-Amz-Target": bcmdTargetPrefix + "ListDashboards",
	}, "s3")
	if body := mustBody(t, resp); resp.StatusCode == http.StatusOK && strings.Contains(string(body), "dashboards") {
		t.Fatalf("s3-signed request was served: %s", body)
	}

	// Oversized and unknown-member bodies are refused with each service's error shape.
	big := `{"Pad":"` + strings.Repeat("x", budMaxBody) + `"}`
	if code := budErrAPI(t, ts, bcmdAPI, "ListDashboards", big, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("oversized: %s", code)
	}
	status, out := budCallAPI(t, ts, bcmpcAPI, "GetPreferences", `{"nope":1}`)
	if status != http.StatusBadRequest || out["__type"] != "ValidationException" || out["reason"] != "fieldValidationFailed" || out["message"] == nil {
		t.Fatalf("unknown member: %d %v", status, out)
	}
	if status, _ := acpCall(t, ts, http.MethodGet, "/reportDefinition?maxResults=0", ""); status != http.StatusBadRequest {
		t.Fatalf("maxResults=0: %d", status)
	}
}

// acpCall sends a signed Application Cost Profiler request and decodes the reply.
func acpCall(t *testing.T, ts *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, map[string]string{"Content-Type": "application/json"}, "application-cost-profiler")
	data := mustBody(t, resp)
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("%s %s: content type %q", method, path, ct)
	}
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, path, data, err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		out["__code"] = resp.Header.Get("X-Amzn-ErrorType")
	}
	return resp.StatusCode, out
}
