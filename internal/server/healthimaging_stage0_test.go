package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func healthImagingRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
	}
	headers := map[string]string{}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "medical-imaging")
}

func TestHealthImagingStage0CatalogCoverage(t *testing.T) {
	if len(healthImagingOperations) != 18 {
		t.Fatalf("expected 18 HealthImaging operations from docs, got %d", len(healthImagingOperations))
	}
	if len(healthImagingOperationByName) != len(healthImagingOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateDatastore",
		"ListDatastores",
		"StartDICOMImportJob",
		"GetImageSet",
		"SearchImageSets",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := healthImagingOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(healthImagingDataTypes) != 22 {
		t.Fatalf("expected 22 HealthImaging data types from docs, got %d", len(healthImagingDataTypes))
	}
	if len(healthImagingDataTypeByName) != len(healthImagingDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"DatastoreProperties",
		"DICOMImportJobProperties",
		"ImageSetProperties",
		"SearchCriteria",
		"Sort",
	}
	for _, typeName := range requiredTypes {
		if _, ok := healthImagingDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestHealthImagingStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := healthImagingRequest(t, ts, http.MethodGet, "/unknown-healthimaging-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestHealthImagingKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := healthImagingRequest(t, ts, http.MethodGet, "/datastore", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "datastorePropertiesList") {
		t.Fatalf("expected ListDatastores response body to include datastorePropertiesList, got %q", body)
	}
}
