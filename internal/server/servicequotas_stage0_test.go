package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serviceQuotasRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "ServiceQuotasV20190624." + action,
		},
		"servicequotas",
	)
}

func TestServiceQuotasStage0CatalogCoverage(t *testing.T) {
	if len(serviceQuotasOperations) != 26 {
		t.Fatalf("expected 26 Service Quotas operations from docs, got %d", len(serviceQuotasOperations))
	}
	if len(serviceQuotasOperationByName) != len(serviceQuotasOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"ListServices",
		"ListServiceQuotas",
		"GetServiceQuota",
		"RequestServiceQuotaIncrease",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := serviceQuotasOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(serviceQuotasDataTypes) != 11 {
		t.Fatalf("expected 11 Service Quotas data types from docs, got %d", len(serviceQuotasDataTypes))
	}
	if len(serviceQuotasDataTypeByName) != len(serviceQuotasDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ServiceInfo",
		"ServiceQuota",
		"RequestedServiceQuotaChange",
		"ServiceQuotaIncreaseRequestInTemplate",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := serviceQuotasDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestServiceQuotasStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := serviceQuotasRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestServiceQuotasKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := serviceQuotasRequest(t, ts, "ListServices", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "Services") {
		t.Fatalf("expected ListServices response body to include Services, got %q", body)
	}
}

func TestServiceQuotasStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range serviceQuotasOperations {
		resp := serviceQuotasRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
