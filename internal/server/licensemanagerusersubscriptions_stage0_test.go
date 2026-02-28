package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func licenseManagerUserSubscriptionsRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "license-manager-user-subscriptions")
}

func licenseManagerUserSubscriptionsPathForTest(template string) string {
	resourceARN := "arn:aws:license-manager-user-subscriptions:us-east-1:123456789012:resource/stackyard-resource"
	out := template
	out = strings.ReplaceAll(out, "{ResourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestLicenseManagerUserSubscriptionsStage0CatalogCoverage(t *testing.T) {
	if len(licenseManagerUserSubscriptionsOperations) != 17 {
		t.Fatalf("expected 17 License Manager User Subscriptions operations from docs, got %d", len(licenseManagerUserSubscriptionsOperations))
	}
	if len(licenseManagerUserSubscriptionsOperationByName) != len(licenseManagerUserSubscriptionsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"RegisterIdentityProvider",
		"CreateLicenseServerEndpoint",
		"AssociateUser",
		"StartProductSubscription",
		"ListInstances",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := licenseManagerUserSubscriptionsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(licenseManagerUserSubscriptionsDataTypes) != 20 {
		t.Fatalf("expected 20 License Manager User Subscriptions data types from docs, got %d", len(licenseManagerUserSubscriptionsDataTypes))
	}
	if len(licenseManagerUserSubscriptionsDataTypeByName) != len(licenseManagerUserSubscriptionsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{"Filter", "IdentityProvider", "InstanceSummary", "LicenseServerEndpoint", "ProductUserSummary", "UpdateSettings"}
	for _, typeName := range requiredTypes {
		if _, ok := licenseManagerUserSubscriptionsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestLicenseManagerUserSubscriptionsStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := licenseManagerUserSubscriptionsRequest(t, ts, http.MethodGet, "/user/UnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestLicenseManagerUserSubscriptionsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := licenseManagerUserSubscriptionsRequest(t, ts, http.MethodPost, "/user/ListProductSubscriptions", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "ProductUserSummaries") {
		t.Fatalf("expected ListProductSubscriptions response body to include ProductUserSummaries, got %q", body)
	}
}

func TestLicenseManagerUserSubscriptionsAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range licenseManagerUserSubscriptionsOperations {
		path := licenseManagerUserSubscriptionsPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := licenseManagerUserSubscriptionsRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
