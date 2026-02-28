package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func awsCostManagementRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	if strings.TrimSpace(payload) == "" {
		payload = `{}`
	}
	headers := map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": "AWSBillingAndCostManagement." + action,
	}
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(payload), headers, "ce")
}

func TestAWSCostManagementStage0CatalogCoverage(t *testing.T) {
	if len(awsCostManagementOperations) != 202 {
		t.Fatalf("expected 202 AWS Billing and Cost Management actions from docs, got %d", len(awsCostManagementOperations))
	}
	if len(awsCostManagementOperationByName) != len(awsCostManagementOperations) {
		t.Fatalf("expected unique AWS Billing and Cost Management action names")
	}

	requiredActions := []string{
		"CreateAnomalyMonitor",
		"GetCostAndUsage",
		"budgets_CreateBudget",
		"cur_PutReportDefinition",
		"pricing_GetProducts",
		"taxSettings_GetTaxRegistration",
		"invoicing_ListInvoiceUnits",
		"billing_ListBillingViews",
		"DataExports_ListExports",
		"CostOptimizationHub_ListRecommendations",
	}
	for _, action := range requiredActions {
		if _, ok := awsCostManagementOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(awsCostManagementDataTypes) != 386 {
		t.Fatalf("expected 386 AWS Billing and Cost Management data types from docs, got %d", len(awsCostManagementDataTypes))
	}
	if len(awsCostManagementDataTypeByName) != len(awsCostManagementDataTypes) {
		t.Fatalf("expected unique AWS Billing and Cost Management data type names")
	}

	requiredTypes := []string{
		"AnomalyMonitor",
		"AnomalySubscription",
		"Expression",
		"budgets_Budget",
		"cur_ReportDefinition",
		"pricing_Filter",
		"taxSettings_TaxRegistration",
		"invoicing_InvoiceUnit",
		"billing_BillingViewElement",
		"DataExports_Export",
		"CostOptimizationHub_Recommendation",
	}
	for _, typeName := range requiredTypes {
		if _, ok := awsCostManagementDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestAWSCostManagementStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": "AWSBillingAndCostManagement.UnknownAction",
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), headers, "ce")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAWSCostManagementStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := awsCostManagementRequest(t, ts, "GetCostAndUsage", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
}

func TestAWSCostManagementStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range awsCostManagementOperations {
		resp := awsCostManagementRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
