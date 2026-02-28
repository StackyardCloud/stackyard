package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAWSCostManagementStage123456LifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:awscostmanagement:us-east-1:123456789012:anomalymonitor/monitor-000001"
	cases := []struct {
		name    string
		action  string
		payload string
	}{
		{name: "CreateAnomalyMonitor", action: "CreateAnomalyMonitor", payload: `{"monitorName":"stage-monitor"}`},
		{name: "CreateAnomalySubscription", action: "CreateAnomalySubscription", payload: `{"subscriptionName":"stage-subscription"}`},
		{name: "GetAnomalyMonitors", action: "GetAnomalyMonitors", payload: `{}`},
		{name: "GetAnomalySubscriptions", action: "GetAnomalySubscriptions", payload: `{}`},
		{name: "GetCostAndUsage", action: "GetCostAndUsage", payload: `{}`},
		{name: "GetCostForecast", action: "GetCostForecast", payload: `{}`},
		{name: "ListCostCategoryDefinitions", action: "ListCostCategoryDefinitions", payload: `{}`},
		{name: "budgets_CreateBudget", action: "budgets_CreateBudget", payload: `{"BudgetName":"stage-budget"}`},
		{name: "budgets_DescribeBudgets", action: "budgets_DescribeBudgets", payload: `{}`},
		{name: "cur_PutReportDefinition", action: "cur_PutReportDefinition", payload: `{"ReportName":"stage-report"}`},
		{name: "cur_DescribeReportDefinitions", action: "cur_DescribeReportDefinitions", payload: `{}`},
		{name: "pricing_GetProducts", action: "pricing_GetProducts", payload: `{}`},
		{name: "taxSettings_GetTaxRegistration", action: "taxSettings_GetTaxRegistration", payload: `{}`},
		{name: "invoicing_ListInvoiceUnits", action: "invoicing_ListInvoiceUnits", payload: `{}`},
		{name: "billing_ListBillingViews", action: "billing_ListBillingViews", payload: `{}`},
		{name: "DataExports_ListExports", action: "DataExports_ListExports", payload: `{}`},
		{name: "CostOptimizationHub_ListRecommendations", action: "CostOptimizationHub_ListRecommendations", payload: `{}`},
		{name: "TagResource", action: "TagResource", payload: `{"resourceArn":"` + resourceARN + `","tags":{"env":"stage","owner":"tests"}}`},
		{name: "ListTagsForResource", action: "ListTagsForResource", payload: `{"resourceArn":"` + resourceARN + `"}`},
		{name: "UntagResource", action: "UntagResource", payload: `{"resourceArn":"` + resourceARN + `","tagKeys":["owner"]}`},
		{name: "DeleteAnomalySubscription", action: "DeleteAnomalySubscription", payload: `{}`},
		{name: "DeleteAnomalyMonitor", action: "DeleteAnomalyMonitor", payload: `{}`},
	}

	for _, tc := range cases {
		resp := awsCostManagementRequest(t, ts, tc.action, tc.payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tc.name, resp.StatusCode, body)
		}
		if strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s: expected non-NotImplemented response, got: %s", tc.name, body)
		}
	}
}
