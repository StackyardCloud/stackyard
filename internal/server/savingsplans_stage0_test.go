package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func savingsPlansRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, []byte(payload), headers, "savingsplans")
}

func TestSavingsPlansStage0CatalogCoverage(t *testing.T) {
	if len(savingsPlansOperations) != 10 {
		t.Fatalf("expected 10 Savings Plans actions from docs, got %d", len(savingsPlansOperations))
	}
	if len(savingsPlansOperationByName) != len(savingsPlansOperations) {
		t.Fatalf("expected unique Savings Plans action names")
	}

	requiredActions := []string{
		"CreateSavingsPlan",
		"DeleteQueuedSavingsPlan",
		"DescribeSavingsPlanRates",
		"DescribeSavingsPlans",
		"DescribeSavingsPlansOfferingRates",
		"DescribeSavingsPlansOfferings",
		"ListTagsForResource",
		"ReturnSavingsPlan",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := savingsPlansOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(savingsPlansTypes) != 12 {
		t.Fatalf("expected 12 Savings Plans data types from docs, got %d", len(savingsPlansTypes))
	}
	if len(savingsPlansTypeByName) != len(savingsPlansTypes) {
		t.Fatalf("expected unique Savings Plans data type names")
	}

	requiredTypes := []string{
		"SavingsPlan",
		"SavingsPlanFilter",
		"SavingsPlanOffering",
		"SavingsPlanOfferingRate",
		"SavingsPlanRate",
	}
	for _, typeName := range requiredTypes {
		if _, ok := savingsPlansTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestSavingsPlansStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := savingsPlansRequest(t, ts, http.MethodPost, "/savingsplans-unknown-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSavingsPlansStage0KnownActionReturnsDescribeSavingsPlans(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := savingsPlansRequest(t, ts, http.MethodPost, "/DescribeSavingsPlans", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "savingsPlans") {
		t.Fatalf("expected DescribeSavingsPlans response body to include savingsPlans, got %q", body)
	}
}

func TestSavingsPlansStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range savingsPlansOperations {
		resp := savingsPlansRequest(t, ts, op.Method, op.URI, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
