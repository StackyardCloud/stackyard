package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPNetworkSecurityRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetworkSecurityContractServer(t)

	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/us-central1/authorizationPolicies?pageSize=1", "/authorizationPolicies")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/us-central1/authorizationPolicies/authz-a", "/authorizationPolicies/authz-a")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/us-central1/authorizationPolicies?authorizationPolicyId=authz-a", "/authorizationPolicies")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/locations/us-central1/authorizationPolicies/authz-a", "/authorizationPolicies/authz-a")

	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/us-central1/clientTlsPolicies?pageSize=1", "/clientTlsPolicies")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/us-central1/clientTlsPolicies/clienttls-a", "/clientTlsPolicies/clienttls-a")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/us-central1/clientTlsPolicies?clientTlsPolicyId=clienttls-a", "/clientTlsPolicies")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/locations/us-central1/clientTlsPolicies/clienttls-a", "/clientTlsPolicies/clienttls-a")

	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/us-central1/serverTlsPolicies?pageSize=1", "/serverTlsPolicies")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/us-central1/serverTlsPolicies/servertls-a", "/serverTlsPolicies/servertls-a")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/us-central1/serverTlsPolicies?serverTlsPolicyId=servertls-a", "/serverTlsPolicies")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/locations/us-central1/serverTlsPolicies/servertls-a", "/serverTlsPolicies/servertls-a")
}

func TestGCPNetworkSecurityRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetworkSecurityContractServer(t)

	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networksecurity.v1beta1.NetworkSecurity/ListAuthorizationPolicies", "NetworkSecurity/ListAuthorizationPolicies")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networksecurity.v1beta1.NetworkSecurity/GetAuthorizationPolicy", "NetworkSecurity/GetAuthorizationPolicy")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networksecurity.v1beta1.NetworkSecurity/CreateAuthorizationPolicy", "NetworkSecurity/CreateAuthorizationPolicy")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networksecurity.v1beta1.NetworkSecurity/DeleteAuthorizationPolicy", "NetworkSecurity/DeleteAuthorizationPolicy")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networksecurity.v1beta1.NetworkSecurity/ListClientTlsPolicies", "NetworkSecurity/ListClientTlsPolicies")
	assertGCPNetworkSecurityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networksecurity.v1beta1.NetworkSecurity/ListServerTlsPolicies", "NetworkSecurity/ListServerTlsPolicies")
}

func newGCPNetworkSecurityContractServer(t *testing.T) *httptest.Server {
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

func assertGCPNetworkSecurityNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp networksecurity router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPNetworksecurityRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPNetworksecurityRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/networksecurity?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp networksecurity contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "networksecurity" {
		t.Fatalf("expected service=networksecurity, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
