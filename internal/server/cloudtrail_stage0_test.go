package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCloudTrailStage0CatalogCoverage(t *testing.T) {
	if len(cloudTrailOperations) != 60 {
		t.Fatalf("expected 60 CloudTrail operations from docs, got %d", len(cloudTrailOperations))
	}
	if len(cloudTrailOperationByName) != len(cloudTrailOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateTrail",
		"DescribeTrails",
		"LookupEvents",
		"StartLogging",
		"StopLogging",
		"GetEventSelectors",
	}
	for _, action := range requiredActions {
		if _, ok := cloudTrailOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(cloudTrailDataTypes) != 36 {
		t.Fatalf("expected 36 CloudTrail data types from docs, got %d", len(cloudTrailDataTypes))
	}
	if len(cloudTrailDataTypeByName) != len(cloudTrailDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Trail",
		"TrailInfo",
		"Event",
		"EventSelector",
		"Tag",
		"Destination",
	}
	for _, typeName := range requiredTypes {
		if _, ok := cloudTrailDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func cloudTrailRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "com.amazonaws.cloudtrail.v20131101.CloudTrail_20131101." + action,
		},
		"cloudtrail",
	)
}

func TestCloudTrailStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudTrailRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCloudTrailStage0KnownActionDoesNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudTrailRequest(t, ts, "DescribeTrails", `{}`)
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected DescribeTrails to be implemented")
	}
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected non-NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "trailList") {
		t.Fatalf("expected DescribeTrails response body to include trailList, got %q", body)
	}
}

func TestCloudTrailStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cloudTrailOperations {
		resp := cloudTrailRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
