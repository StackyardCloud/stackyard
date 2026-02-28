package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func billingConductorRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
	}
	headers := map[string]string{}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "billingconductor")
}

func TestBillingConductorStage0CatalogCoverage(t *testing.T) {
	if len(billingConductorOperations) != 32 {
		t.Fatalf("expected 32 Billing Conductor actions from docs, got %d", len(billingConductorOperations))
	}
	if len(billingConductorOperationByName) != len(billingConductorOperations) {
		t.Fatalf("expected unique Billing Conductor action names")
	}

	requiredActions := []string{
		"CreateBillingGroup",
		"CreatePricingPlan",
		"CreatePricingRule",
		"CreateCustomLineItem",
		"AssociateAccounts",
		"AssociatePricingRules",
		"BatchAssociateResourcesToCustomLineItem",
		"GetBillingGroupCostReport",
		"ListBillingGroups",
		"TagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := billingConductorOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(billingConductorDataTypes) != 48 {
		t.Fatalf("expected 48 Billing Conductor data types from docs, got %d", len(billingConductorDataTypes))
	}
	if len(billingConductorDataTypeByName) != len(billingConductorDataTypes) {
		t.Fatalf("expected unique Billing Conductor data type names")
	}

	requiredTypes := []string{
		"BillingGroupListElement",
		"PricingPlanListElement",
		"PricingRuleListElement",
		"CustomLineItemListElement",
		"ListResourcesAssociatedToCustomLineItemResponseElement",
		"UpdateCustomLineItemChargeDetails",
		"UpdatePricingRule",
	}
	for _, typeName := range requiredTypes {
		if _, ok := billingConductorDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestBillingConductorStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := billingConductorRequest(t, ts, http.MethodPost, "/unknown-billingconductor-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestBillingConductorStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := billingConductorRequest(t, ts, http.MethodPost, "/list-billing-groups", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
}

func TestBillingConductorStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range billingConductorOperations {
		path := billingConductorRenderTestURI(op.URI)
		payload := ""
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch || op.Method == http.MethodDelete {
			payload = `{}`
		}
		resp := billingConductorRequest(t, ts, op.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s (path=%s)", op.Name, resp.StatusCode, body, path)
		}
	}
}

var billingConductorURIPlaceholderPattern = regexp.MustCompile(`\{([^}]+)\}`)

func billingConductorRenderTestURI(uriTemplate string) string {
	return billingConductorURIPlaceholderPattern.ReplaceAllStringFunc(uriTemplate, func(raw string) string {
		placeholder := strings.TrimSpace(strings.Trim(raw, "{}"))
		switch strings.ToLower(placeholder) {
		case "resourcearn":
			return url.PathEscape("arn:aws:billingconductor:us-east-1:123456789012:billinggroup/bg-000001")
		default:
			return "stackyard"
		}
	})
}
