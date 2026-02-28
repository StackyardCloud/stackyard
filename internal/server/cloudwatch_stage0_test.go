package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCloudWatchStage0CatalogCoverage(t *testing.T) {
	if len(cloudWatchOperations) != 43 {
		t.Fatalf("expected 43 CloudWatch operations from docs, got %d", len(cloudWatchOperations))
	}
	if len(cloudWatchOperationByName) != len(cloudWatchOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"PutMetricData",
		"GetMetricData",
		"PutMetricAlarm",
		"DescribeAlarms",
		"GetMetricStatistics",
		"PutDashboard",
	}
	for _, action := range requiredActions {
		if _, ok := cloudWatchOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(cloudWatchDataTypes) != 43 {
		t.Fatalf("expected 43 CloudWatch data types from docs, got %d", len(cloudWatchDataTypes))
	}
	if len(cloudWatchDataTypeByName) != len(cloudWatchDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Metric",
		"MetricDatum",
		"MetricDataQuery",
		"MetricDataResult",
		"MetricAlarm",
		"DashboardEntry",
	}
	for _, typeName := range requiredTypes {
		if _, ok := cloudWatchDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func cloudWatchRequest(t *testing.T, ts *httptest.Server, action string, params url.Values) *http.Response {
	t.Helper()
	if params == nil {
		params = url.Values{}
	}
	params.Set("Action", action)
	params.Set("Version", "2010-08-01")
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(params.Encode()),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		"monitoring",
	)
}

func TestCloudWatchStage0UnknownActionReturnsInvalidAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchRequest(t, ts, "TotallyUnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "InvalidAction") {
		t.Fatalf("expected InvalidAction response body, got %q", body)
	}
}

func TestCloudWatchKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchRequest(t, ts, "DescribeAlarms", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "DescribeAlarmsResponse") {
		t.Fatalf("expected DescribeAlarmsResponse in body, got %q", body)
	}
}

func TestCloudWatchAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cloudWatchOperations {
		resp := cloudWatchRequest(t, ts, op.Name, nil)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
