package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func computeOptimizerRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "ComputeOptimizerService." + action,
		},
		"compute-optimizer",
	)
}

func TestComputeOptimizerStage0CatalogCoverage(t *testing.T) {
	if len(computeOptimizerOperations) != 51 {
		t.Fatalf("expected 51 Compute Optimizer operations from docs, got %d", len(computeOptimizerOperations))
	}
	if len(computeOptimizerOperationByName) != len(computeOptimizerOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"GetEnrollmentStatus",
		"UpdateEnrollmentStatus",
		"GetRecommendationSummaries",
		"DescribeRecommendationExportJobs",
		"PutRecommendationPreferences",
		"automation_CreateAutomationRule",
	}
	for _, action := range requiredActions {
		if _, ok := computeOptimizerOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(computeOptimizerDataTypes) != 135 {
		t.Fatalf("expected 135 Compute Optimizer data types from docs, got %d", len(computeOptimizerDataTypes))
	}
	if len(computeOptimizerDataTypeByName) != len(computeOptimizerDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AccountEnrollmentStatus",
		"InstanceRecommendation",
		"RecommendationSummary",
		"UtilizationMetric",
		"automation_AutomationRule",
		"automation_RecommendedAction",
	}
	for _, typeName := range requiredTypes {
		if _, ok := computeOptimizerDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestComputeOptimizerStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := computeOptimizerRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestComputeOptimizerKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := computeOptimizerRequest(t, ts, "GetEnrollmentStatus", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
}

func TestComputeOptimizerStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range computeOptimizerOperations {
		resp := computeOptimizerRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
