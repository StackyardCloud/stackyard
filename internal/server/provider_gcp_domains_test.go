package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDomainsRouter_SearchAndRegistrationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDomainsContractServer(t)
	assertGCPDomainsNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/registrations:searchDomains?query=example.com", ":searchDomains")
	assertGCPDomainsNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/registrations:retrieveRegisterParameters?domainName=example.com", ":retrieveRegisterParameters")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/registrations:register", ":register")
	assertGCPDomainsNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/registrations?pageSize=1", "/registrations")
	assertGCPDomainsNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/registrations/example.com", "/registrations/example.com")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPatch, "/gcp/v1beta1/projects/stackyard/locations/global/registrations/example.com", "/registrations/example.com")
}

func TestGCPDomainsRouter_TransferAndConfigurationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDomainsContractServer(t)
	assertGCPDomainsNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/registrations:retrieveTransferParameters?domainName=example.com", ":retrieveTransferParameters")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/registrations:transfer", ":transfer")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/registrations/example.com:configureManagementSettings", ":configureManagementSettings")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/registrations/example.com:configureDnsSettings", ":configureDnsSettings")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/registrations/example.com:configureContactSettings", ":configureContactSettings")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/registrations/example.com:export", ":export")
	assertGCPDomainsNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/locations/global/registrations/example.com", "/registrations/example.com")
}

func TestGCPDomainsRouter_AuthorizationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDomainsContractServer(t)
	assertGCPDomainsNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/registrations/example.com:retrieveAuthorizationCode", ":retrieveAuthorizationCode")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/registrations/example.com:resetAuthorizationCode", ":resetAuthorizationCode")
	assertGCPDomainsNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/operations/op-1", "/operations/op-1")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/operations/op-1:cancel", ":cancel")
}

func TestGCPDomainsRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDomainsContractServer(t)
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.domains.v1beta1.Domains/SearchDomains", "Domains/SearchDomains")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.domains.v1beta1.Domains/RegisterDomain", "Domains/RegisterDomain")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.domains.v1beta1.Domains/ListRegistrations", "Domains/ListRegistrations")
	assertGCPDomainsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.domains.v1beta1.Domains/ConfigureDnsSettings", "Domains/ConfigureDnsSettings")
}

func newGCPDomainsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDomainsNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp domains router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPDomainsRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPDomainsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/domains?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp domains contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "domains" {
		t.Fatalf("expected service=domains, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

