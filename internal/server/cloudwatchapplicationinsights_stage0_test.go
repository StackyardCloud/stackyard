package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cloudWatchApplicationInsightsRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "EC2WindowsBarleyService." + action,
		},
		"applicationinsights",
	)
}

func TestCloudWatchApplicationInsightsStage0CatalogCoverage(t *testing.T) {
	if len(cloudWatchApplicationInsightsOperations) != 33 {
		t.Fatalf("expected 33 CloudWatch Application Insights operations from docs, got %d", len(cloudWatchApplicationInsightsOperations))
	}
	if len(cloudWatchApplicationInsightsOperationByName) != len(cloudWatchApplicationInsightsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateApplication",
		"DescribeApplication",
		"ListApplications",
		"CreateComponent",
		"ListProblems",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := cloudWatchApplicationInsightsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(cloudWatchApplicationInsightsDataTypes) != 10 {
		t.Fatalf("expected 10 CloudWatch Application Insights data types from docs, got %d", len(cloudWatchApplicationInsightsDataTypes))
	}
	if len(cloudWatchApplicationInsightsDataTypeByName) != len(cloudWatchApplicationInsightsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ApplicationInfo",
		"ApplicationComponent",
		"LogPattern",
		"Problem",
		"Observation",
	}
	for _, typeName := range requiredTypes {
		if _, ok := cloudWatchApplicationInsightsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCloudWatchApplicationInsightsStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchApplicationInsightsRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCloudWatchApplicationInsightsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchApplicationInsightsRequest(t, ts, "ListApplications", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "ApplicationInfoList") {
		t.Fatalf("expected ListApplications response body to include ApplicationInfoList, got %q", body)
	}
}

func TestCloudWatchApplicationInsightsAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cloudWatchApplicationInsightsOperations {
		resp := cloudWatchApplicationInsightsRequest(t, ts, op.Name, `{}`)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
