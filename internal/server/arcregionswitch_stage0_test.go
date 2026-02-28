package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func arcRegionSwitchRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "ArcRegionSwitchService." + action,
		},
		"arc-region-switch",
	)
}

func TestARCRegionSwitchStage0CatalogCoverage(t *testing.T) {
	if len(arcRegionSwitchOperations) != 21 {
		t.Fatalf("expected 21 ARC Region Switch operations from docs, got %d", len(arcRegionSwitchOperations))
	}
	if len(arcRegionSwitchOperationByName) != len(arcRegionSwitchOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreatePlan",
		"GetPlan",
		"ListPlans",
		"StartPlanExecution",
		"GetPlanExecution",
		"ListPlanExecutions",
		"UpdatePlanExecutionStep",
		"ListRoute53HealthChecks",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := arcRegionSwitchOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(arcRegionSwitchDataTypes) != 46 {
		t.Fatalf("expected 46 ARC Region Switch data types from docs, got %d", len(arcRegionSwitchDataTypes))
	}
	if len(arcRegionSwitchDataTypeByName) != len(arcRegionSwitchDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Plan",
		"AbbreviatedPlan",
		"AbbreviatedExecution",
		"ExecutionEvent",
		"Route53HealthCheck",
		"Workflow",
	}
	for _, typeName := range requiredTypes {
		if _, ok := arcRegionSwitchDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestARCRegionSwitchStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := arcRegionSwitchRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestARCRegionSwitchStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := arcRegionSwitchRequest(t, ts, "ListPlans", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "plans") {
		t.Fatalf("expected ListPlans response body to include plans, got %q", body)
	}
}

func TestARCRegionSwitchStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range arcRegionSwitchOperations {
		resp := arcRegionSwitchRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
