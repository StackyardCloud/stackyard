package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func healthLakeRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "HealthLake." + action,
		},
		"healthlake",
	)
}

func TestHealthLakeStage0CatalogCoverage(t *testing.T) {
	if len(healthLakeOperations) != 13 {
		t.Fatalf("expected 13 HealthLake operations from docs, got %d", len(healthLakeOperations))
	}
	if len(healthLakeOperationByName) != len(healthLakeOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateFHIRDatastore",
		"DescribeFHIRDatastore",
		"ListFHIRDatastores",
		"StartFHIRImportJob",
		"StartFHIRExportJob",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := healthLakeOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(healthLakeDataTypes) != 14 {
		t.Fatalf("expected 14 HealthLake data types from docs, got %d", len(healthLakeDataTypes))
	}
	if len(healthLakeDataTypeByName) != len(healthLakeDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"DatastoreFilter",
		"DatastoreProperties",
		"ImportJobProperties",
		"ExportJobProperties",
		"S3Configuration",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := healthLakeDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestHealthLakeStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := healthLakeRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestHealthLakeKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := healthLakeRequest(t, ts, "ListFHIRDatastores", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "DatastorePropertiesList") {
		t.Fatalf("expected ListFHIRDatastores response body to include DatastorePropertiesList, got %q", body)
	}
}
