package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func healthRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSHealth_20160804." + action,
		},
		"health",
	)
}

func TestHealthStage0CatalogCoverage(t *testing.T) {
	if len(healthOperations) != 14 {
		t.Fatalf("expected 14 Health operations from docs, got %d", len(healthOperations))
	}
	if len(healthOperationByName) != len(healthOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"DescribeEvents",
		"DescribeEventTypes",
		"DescribeEventDetails",
		"DescribeAffectedEntities",
		"DescribeAffectedAccountsForOrganization",
		"EnableHealthServiceAccessForOrganization",
	}
	for _, action := range requiredActions {
		if _, ok := healthOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(healthDataTypes) != 22 {
		t.Fatalf("expected 22 Health data types from docs, got %d", len(healthDataTypes))
	}
	if len(healthDataTypeByName) != len(healthDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AffectedEntity",
		"EntityAggregate",
		"Event",
		"EventDetails",
		"EventType",
		"OrganizationEvent",
	}
	for _, typeName := range requiredTypes {
		if _, ok := healthDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestHealthStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := healthRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestHealthKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := healthRequest(t, ts, "DescribeEvents", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "events") {
		t.Fatalf("expected DescribeEvents response body to include events, got %q", body)
	}
}

func TestHealthStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range healthOperations {
		resp := healthRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
