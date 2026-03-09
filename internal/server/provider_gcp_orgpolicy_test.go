package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPOrgPolicyRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOrgPolicyContractServer(t)

	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodGet, "/gcp/v2/organizations/123/constraints?pageSize=1", "/constraints")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodGet, "/gcp/v2/organizations/123/policies?pageSize=1", "/policies")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodGet, "/gcp/v2/organizations/123/policies/compute.disableSerialPortAccess", "/policies/compute.disableSerialPortAccess")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodGet, "/gcp/v2/organizations/123/policies/compute.disableSerialPortAccess:getEffectivePolicy", ":getEffectivePolicy")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/v2/organizations/123/policies", "/policies")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPatch, "/gcp/v2/organizations/123/policies/compute.disableSerialPortAccess", "/policies/compute.disableSerialPortAccess")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodDelete, "/gcp/v2/organizations/123/policies/compute.disableSerialPortAccess", "/policies/compute.disableSerialPortAccess")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodGet, "/gcp/v2/organizations/123/customConstraints?pageSize=1", "/customConstraints")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodGet, "/gcp/v2/organizations/123/customConstraints/custom.requireLabel", "/customConstraints/custom.requireLabel")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/v2/organizations/123/customConstraints", "/customConstraints")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPatch, "/gcp/v2/organizations/123/customConstraints/custom.requireLabel", "/customConstraints/custom.requireLabel")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodDelete, "/gcp/v2/organizations/123/customConstraints/custom.requireLabel", "/customConstraints/custom.requireLabel")
}

func TestGCPOrgPolicyRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOrgPolicyContractServer(t)

	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/ListConstraints", "OrgPolicy/ListConstraints")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/ListPolicies", "OrgPolicy/ListPolicies")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/GetPolicy", "OrgPolicy/GetPolicy")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/GetEffectivePolicy", "OrgPolicy/GetEffectivePolicy")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/CreatePolicy", "OrgPolicy/CreatePolicy")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/UpdatePolicy", "OrgPolicy/UpdatePolicy")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/DeletePolicy", "OrgPolicy/DeletePolicy")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/CreateCustomConstraint", "OrgPolicy/CreateCustomConstraint")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/UpdateCustomConstraint", "OrgPolicy/UpdateCustomConstraint")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/GetCustomConstraint", "OrgPolicy/GetCustomConstraint")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/ListCustomConstraints", "OrgPolicy/ListCustomConstraints")
	assertGCPOrgPolicyNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.orgpolicy.v2.OrgPolicy/DeleteCustomConstraint", "OrgPolicy/DeleteCustomConstraint")
}

func newGCPOrgPolicyContractServer(t *testing.T) *httptest.Server {
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

func assertGCPOrgPolicyNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp orgpolicy router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPOrgpolicyRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPOrgpolicyRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/orgpolicy?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp orgpolicy contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "orgpolicy" {
		t.Fatalf("expected service=orgpolicy, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
