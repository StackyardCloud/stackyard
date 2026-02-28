package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func directoryServiceRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "DirectoryService_20150416." + action,
		},
		"ds",
	)
}

func TestDirectoryServiceStage0CatalogCoverage(t *testing.T) {
	if len(directoryServiceOperations) != 70 {
		t.Fatalf("expected 70 Directory Service operations from docs, got %d", len(directoryServiceOperations))
	}
	if len(directoryServiceOperationByName) != len(directoryServiceOperations) {
		t.Fatalf("expected unique Directory Service operation names")
	}

	requiredActions := []string{
		"CreateDirectory",
		"DescribeDirectories",
		"CreateSnapshot",
		"DescribeSnapshots",
		"CreateTrust",
		"DescribeTrusts",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := directoryServiceOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(directoryServiceDataTypes) != 213 {
		t.Fatalf("expected 213 Directory Service data types from docs, got %d", len(directoryServiceDataTypes))
	}
	if len(directoryServiceDataTypeByName) != len(directoryServiceDataTypes) {
		t.Fatalf("expected unique Directory Service data type names")
	}

	requiredTypes := []string{
		"DirectoryDescription",
		"Snapshot",
		"Trust",
		"Tag",
		"DirectoryLimits",
		"CertificateInfo",
	}
	for _, typeName := range requiredTypes {
		if _, ok := directoryServiceDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDirectoryServiceStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := directoryServiceRequest(t, ts, "UnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDirectoryServiceStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := directoryServiceRequest(t, ts, "DescribeDirectories", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "DirectoryDescriptions") {
		t.Fatalf("expected response body to include DirectoryDescriptions, got %q", body)
	}
}

func TestDirectoryServiceStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range directoryServiceOperations {
		resp := directoryServiceRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
