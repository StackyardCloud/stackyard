package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func marketplaceRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"aws-marketplace",
	)
}

func TestMarketplaceStage0CatalogCoverage(t *testing.T) {
	if len(marketplaceOperations) != 23 {
		t.Fatalf("expected 23 AWS Marketplace operations from docs, got %d", len(marketplaceOperations))
	}
	if len(marketplaceOperationByName) != len(marketplaceOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"StartChangeSet",
		"DescribeAgreement",
		"GetEntitlements",
		"GetBuyerDashboard",
	}
	for _, action := range requiredActions {
		if _, ok := marketplaceOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(marketplaceDataTypes) != 140 {
		t.Fatalf("expected 140 AWS Marketplace data types from docs, got %d", len(marketplaceDataTypes))
	}
	if len(marketplaceDataTypeByName) != len(marketplaceDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"EntitySummary",
		"AgreementViewSummary",
		"UsageRecord",
		"Entitlement",
		"DeploymentParameterInput",
	}
	for _, typeName := range requiredTypes {
		if _, ok := marketplaceDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestMarketplaceStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := marketplaceRequest(t, ts, http.MethodPost, "/marketplace/unknown-action", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMarketplaceKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := marketplaceRequest(t, ts, http.MethodPost, "/marketplace/list-entities", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "EntitySummaryList") {
		t.Fatalf("expected ListEntities response body to include EntitySummaryList, got %q", body)
	}
}

func TestMarketplaceStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range marketplaceOperations {
		payload := "{}"
		if op.Method == http.MethodGet || op.Method == http.MethodDelete {
			payload = ""
		}
		resp := marketplaceRequest(t, ts, op.Method, op.URI, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
