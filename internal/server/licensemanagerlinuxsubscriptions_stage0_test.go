package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func licenseManagerLinuxSubscriptionsRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "license-manager-linux-subscriptions")
}

func licenseManagerLinuxSubscriptionsPathForTest(template string) string {
	resourceARN := "arn:aws:license-manager-linux-subscriptions:us-east-1:123456789012:resource/stackyard-resource"
	out := template
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestLicenseManagerLinuxSubscriptionsStage0CatalogCoverage(t *testing.T) {
	if len(licenseManagerLinuxSubscriptionsOperations) != 11 {
		t.Fatalf("expected 11 License Manager Linux Subscriptions operations from docs, got %d", len(licenseManagerLinuxSubscriptionsOperations))
	}
	if len(licenseManagerLinuxSubscriptionsOperationByName) != len(licenseManagerLinuxSubscriptionsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"RegisterSubscriptionProvider",
		"GetRegisteredSubscriptionProvider",
		"ListLinuxSubscriptions",
		"GetServiceSettings",
		"TagResource",
		"UpdateServiceSettings",
	}
	for _, action := range requiredActions {
		if _, ok := licenseManagerLinuxSubscriptionsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(licenseManagerLinuxSubscriptionsDataTypes) != 6 {
		t.Fatalf("expected 6 License Manager Linux Subscriptions data types from docs, got %d", len(licenseManagerLinuxSubscriptionsDataTypes))
	}
	if len(licenseManagerLinuxSubscriptionsDataTypeByName) != len(licenseManagerLinuxSubscriptionsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{"Filter", "Instance", "LinuxSubscriptionsDiscoverySettings", "RegisteredSubscriptionProvider", "Subscription", "UpdateServiceSettings"}
	for _, typeName := range requiredTypes {
		if _, ok := licenseManagerLinuxSubscriptionsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestLicenseManagerLinuxSubscriptionsStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := licenseManagerLinuxSubscriptionsRequest(t, ts, http.MethodGet, "/subscription/UnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestLicenseManagerLinuxSubscriptionsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := licenseManagerLinuxSubscriptionsRequest(t, ts, http.MethodPost, "/subscription/ListLinuxSubscriptions", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Subscriptions") {
		t.Fatalf("expected ListLinuxSubscriptions response body to include Subscriptions, got %q", body)
	}
}

func TestLicenseManagerLinuxSubscriptionsAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range licenseManagerLinuxSubscriptionsOperations {
		path := licenseManagerLinuxSubscriptionsPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := licenseManagerLinuxSubscriptionsRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
