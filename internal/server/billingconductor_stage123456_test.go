package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBillingConductorStage123456LifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:billingconductor:us-east-1:123456789012:billinggroup/bg-000001"
	tagPath := "/tags/" + url.PathEscape(resourceARN)

	cases := []struct {
		name    string
		method  string
		path    string
		payload string
	}{
		{name: "CreatePricingRule", method: http.MethodPost, path: "/create-pricing-rule", payload: `{"name":"stage-pricing-rule"}`},
		{name: "CreatePricingPlan", method: http.MethodPost, path: "/create-pricing-plan", payload: `{"name":"stage-pricing-plan"}`},
		{name: "AssociatePricingRules", method: http.MethodPut, path: "/associate-pricing-rules", payload: `{"pricingPlanArn":"arn:aws:billingconductor:us-east-1:123456789012:pricingplan/pp-000001","pricingRuleArns":["arn:aws:billingconductor:us-east-1:123456789012:pricingrule/pr-000001"]}`},
		{name: "CreateBillingGroup", method: http.MethodPost, path: "/create-billing-group", payload: `{"name":"stage-billing-group"}`},
		{name: "AssociateAccounts", method: http.MethodPost, path: "/associate-accounts", payload: `{"billingGroupArn":"arn:aws:billingconductor:us-east-1:123456789012:billinggroup/bg-000001","accountIds":["111122223333"]}`},
		{name: "CreateCustomLineItem", method: http.MethodPost, path: "/create-custom-line-item", payload: `{"name":"stage-custom-line-item"}`},
		{name: "BatchAssociateResourcesToCustomLineItem", method: http.MethodPut, path: "/batch-associate-resources-to-custom-line-item", payload: `{"customLineItemArn":"arn:aws:billingconductor:us-east-1:123456789012:customlineitem/cli-000001","resourceArns":["arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0"]}`},
		{name: "GetBillingGroupCostReport", method: http.MethodPost, path: "/get-billing-group-cost-report", payload: `{"billingGroupArn":"arn:aws:billingconductor:us-east-1:123456789012:billinggroup/bg-000001"}`},
		{name: "ListBillingGroups", method: http.MethodPost, path: "/list-billing-groups", payload: `{}`},
		{name: "ListPricingPlans", method: http.MethodPost, path: "/list-pricing-plans", payload: `{}`},
		{name: "ListPricingRules", method: http.MethodPost, path: "/list-pricing-rules", payload: `{}`},
		{name: "ListCustomLineItems", method: http.MethodPost, path: "/list-custom-line-items", payload: `{}`},
		{name: "ListResourcesAssociatedToCustomLineItem", method: http.MethodPost, path: "/list-resources-associated-to-custom-line-item", payload: `{"customLineItemArn":"arn:aws:billingconductor:us-east-1:123456789012:customlineitem/cli-000001"}`},
		{name: "TagResource", method: http.MethodPost, path: tagPath, payload: `{"tags":{"env":"stage","owner":"tests"}}`},
		{name: "ListTagsForResource", method: http.MethodGet, path: tagPath, payload: ``},
		{name: "UntagResource", method: http.MethodDelete, path: tagPath, payload: `{"tagKeys":["owner"]}`},
		{name: "DeleteCustomLineItem", method: http.MethodPost, path: "/delete-custom-line-item", payload: `{"arn":"arn:aws:billingconductor:us-east-1:123456789012:customlineitem/cli-000001"}`},
		{name: "DeleteBillingGroup", method: http.MethodPost, path: "/delete-billing-group", payload: `{"arn":"arn:aws:billingconductor:us-east-1:123456789012:billinggroup/bg-000001"}`},
		{name: "DeletePricingPlan", method: http.MethodPost, path: "/delete-pricing-plan", payload: `{"arn":"arn:aws:billingconductor:us-east-1:123456789012:pricingplan/pp-000001"}`},
		{name: "DeletePricingRule", method: http.MethodPost, path: "/delete-pricing-rule", payload: `{"arn":"arn:aws:billingconductor:us-east-1:123456789012:pricingrule/pr-000001"}`},
	}

	for _, tc := range cases {
		resp := billingConductorRequest(t, ts, tc.method, tc.path, tc.payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tc.name, resp.StatusCode, body)
		}
		if strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s: expected non-NotImplemented response, got: %s", tc.name, body)
		}
	}
}
