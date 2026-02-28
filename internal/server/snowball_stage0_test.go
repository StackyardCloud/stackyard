package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func snowballRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSIESnowballJobManagementService." + action,
		},
		"snowball",
	)
}

func TestSnowballStage0CatalogCoverage(t *testing.T) {
	if len(snowballOperations) != 27 {
		t.Fatalf("expected 27 Snowball operations from docs, got %d", len(snowballOperations))
	}
	if len(snowballOperationByName) != len(snowballOperations) {
		t.Fatalf("expected unique Snowball operation names")
	}

	requiredActions := []string{
		"CreateAddress",
		"CreateCluster",
		"CreateJob",
		"DescribeJob",
		"ListJobs",
		"GetJobManifest",
		"UpdateJobShipmentState",
	}
	for _, action := range requiredActions {
		if _, ok := snowballOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(snowballDataTypes) != 32 {
		t.Fatalf("expected 32 Snowball data types from docs, got %d", len(snowballDataTypes))
	}
	if len(snowballDataTypeByName) != len(snowballDataTypes) {
		t.Fatalf("expected unique Snowball data type names")
	}

	requiredTypes := []string{
		"Address",
		"ClusterMetadata",
		"JobMetadata",
		"LongTermPricingListEntry",
		"ServiceVersion",
	}
	for _, typeName := range requiredTypes {
		if _, ok := snowballDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestSnowballStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := snowballRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSnowballStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := snowballRequest(t, ts, "ListJobs", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "JobListEntries") {
		t.Fatalf("expected ListJobs response body to include JobListEntries, got %q", body)
	}
}

func TestSnowballStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range snowballOperations {
		resp := snowballRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
