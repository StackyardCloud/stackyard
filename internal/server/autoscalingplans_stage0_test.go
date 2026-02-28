package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func autoScalingPlansRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AnyScaleScalingPlannerFrontendService." + action,
		},
		"autoscaling-plans",
	)
}

func TestAutoScalingPlansStage0CatalogCoverage(t *testing.T) {
	if len(autoScalingPlansOperations) != 6 {
		t.Fatalf("expected 6 Auto Scaling Plans operations from docs, got %d", len(autoScalingPlansOperations))
	}
	if len(autoScalingPlansOperationByName) != len(autoScalingPlansOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateScalingPlan",
		"DeleteScalingPlan",
		"DescribeScalingPlanResources",
		"DescribeScalingPlans",
		"GetScalingPlanResourceForecastData",
		"UpdateScalingPlan",
	}
	for _, action := range requiredActions {
		if _, ok := autoScalingPlansOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(autoScalingPlansDataTypes) != 14 {
		t.Fatalf("expected 14 Auto Scaling Plans data types from docs, got %d", len(autoScalingPlansDataTypes))
	}
	if len(autoScalingPlansDataTypeByName) != len(autoScalingPlansDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ApplicationSource",
		"Datapoint",
		"ScalingInstruction",
		"ScalingPlan",
		"ScalingPlanResource",
		"TargetTrackingConfiguration",
	}
	for _, typeName := range requiredTypes {
		if _, ok := autoScalingPlansDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestAutoScalingPlansStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := autoScalingPlansRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAutoScalingPlansKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := autoScalingPlansRequest(t, ts, "DescribeScalingPlans", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "ScalingPlans") {
		t.Fatalf("expected DescribeScalingPlans response body to include ScalingPlans, got %q", body)
	}
}

func TestAutoScalingPlansStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range autoScalingPlansOperations {
		resp := autoScalingPlansRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
