package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func licenseManagerRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSLicenseManager." + action,
		},
		"license-manager",
	)
}

func TestLicenseManagerStage0CatalogCoverage(t *testing.T) {
	if len(licenseManagerOperations) != 62 {
		t.Fatalf("expected 62 License Manager operations from docs, got %d", len(licenseManagerOperations))
	}
	if len(licenseManagerOperationByName) != len(licenseManagerOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateLicenseConfiguration",
		"GetLicenseConfiguration",
		"ListLicenseConfigurations",
		"TagResource",
		"UntagResource",
		"UpdateServiceSettings",
	}
	for _, action := range requiredActions {
		if _, ok := licenseManagerOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(licenseManagerDataTypes) != 58 {
		t.Fatalf("expected 58 License Manager data types from docs, got %d", len(licenseManagerDataTypes))
	}
	if len(licenseManagerDataTypeByName) != len(licenseManagerDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"License",
		"LicenseConfiguration",
		"LicenseUsage",
		"ReportGenerator",
		"TokenData",
		"UpdateServiceSettings",
	}
	for _, typeName := range requiredTypes {
		if _, ok := licenseManagerDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestLicenseManagerStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := licenseManagerRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestLicenseManagerKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := licenseManagerRequest(t, ts, "ListLicenses", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "Licenses") {
		t.Fatalf("expected ListLicenses response body to include Licenses, got %q", body)
	}
}

func TestLicenseManagerStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range licenseManagerOperations {
		resp := licenseManagerRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
