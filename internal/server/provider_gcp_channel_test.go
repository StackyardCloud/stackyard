package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCloudChannelRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudChannelContractServer(t)
	account := "/gcp/v1/accounts/stackyard-account"

	assertGCPCloudChannelSuccess(t, ts, http.MethodGet, account+"/customers?pageSize=1", nil, "customers")
	assertGCPCloudChannelSuccess(t, ts, http.MethodGet, account+"/customers/team-customer", nil, "team-customer")
	assertGCPCloudChannelSuccess(t, ts, http.MethodPost, account+":checkCloudIdentityAccountsExist", []byte(`{"domain":"example.com"}`), "cloudIdentityAccounts")
	assertGCPCloudChannelSuccess(t, ts, http.MethodGet, "/gcp/v1/products?account=accounts/stackyard-account&pageSize=1", nil, "products")
	assertGCPCloudChannelSuccess(t, ts, http.MethodGet, account+"/offers?pageSize=1", nil, "offers")
	assertGCPCloudChannelSuccess(t, ts, http.MethodGet, account+"/customers/team-customer/entitlements?pageSize=1", nil, "entitlements")
	assertGCPCloudChannelSuccess(t, ts, http.MethodGet, account+"/reports?pageSize=1", nil, "reports")
	assertGCPCloudChannelSuccess(t, ts, http.MethodPost, account+"/reports/613bf59q:run", []byte(`{}`), "operations")
	assertGCPCloudChannelSuccess(t, ts, http.MethodGet, account+"/reportJobs/team-report-job:fetchReportResults?pageSize=1", nil, "reportResults")
}

func TestGCPCloudChannelRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudChannelContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.channel.v1.CloudChannelService/ListCustomers", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp cloud channel router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, "CloudChannelService/ListCustomers") {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudChannelRouter_ListProductsRequiresAccount(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudChannelContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/products?pageSize=1", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloud channel router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudChannelRouter_CheckCloudIdentityRequiresDomain(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudChannelContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/accounts/stackyard-account:checkCloudIdentityAccountsExist", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloud channel router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPCloudChannelContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPCloudChannelSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp cloud channel router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
