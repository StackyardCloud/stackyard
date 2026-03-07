package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCloudControlsPartnerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudControlsPartnerContractServer(t)
	base := "/gcp/v1/organizations/123456789/locations/us-central1"

	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodGet, base+"/customers?pageSize=1", nil, "customers")
	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodGet, base+"/customers/team-customer", nil, "team-customer")
	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodPost, base+"/customers?customerId=team-customer", []byte(`{"customer":{"displayName":"Team Customer"}}`), "operations")
	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodPatch, base+"/customers/team-customer", []byte(`{"customer":{"name":"organizations/123456789/locations/us-central1/customers/team-customer"}}`), "operations")
	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodDelete, base+"/customers/team-customer", nil, "operations")

	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodGet, base+"/customers/team-customer/workloads?pageSize=1", nil, "workloads")
	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodGet, base+"/customers/team-customer/workloads/team-workload", nil, "team-workload")
	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodGet, base+"/customers/team-customer/workloads/team-workload/ekmConnections", nil, "ekmConnections")
	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodGet, base+"/customers/team-customer/workloads/team-workload/partnerPermissions", nil, "partnerPermissions")
	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodGet, base+"/customers/team-customer/workloads/team-workload/accessApprovalRequests?pageSize=1", nil, "accessApprovalRequests")
	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodGet, base+"/partner", nil, "partner")
	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodGet, base+"/customers/team-customer/workloads/team-workload/violations?pageSize=1", nil, "violations")
	assertGCPCloudControlsPartnerSuccess(t, ts, http.MethodGet, base+"/customers/team-customer/workloads/team-workload/violations/violation-1", nil, "violation-1")
}

func TestGCPCloudControlsPartnerRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudControlsPartnerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.cloudcontrolspartner.v1.CloudControlsPartnerCore/ListCustomers", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp cloud controls partner router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, "CloudControlsPartnerCore/ListCustomers") {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudControlsPartnerRouter_ListCustomersInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudControlsPartnerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/us-central1/customers?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloud controls partner router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudControlsPartnerRouter_CreateCustomerRequiresBody(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudControlsPartnerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/organizations/123456789/locations/us-central1/customers?customerId=team-customer", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloud controls partner router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPCloudControlsPartnerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPCloudControlsPartnerSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp cloud controls partner router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
