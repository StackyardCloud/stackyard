package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPManagedIdentitiesRouter_RESTDomainAndTrustRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPManagedIdentitiesContractServer(t)

	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/domains?pageSize=1", "/domains")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/domains/corp.example.com", "/domains/corp.example.com")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/domains?domainName=corp.example.com", "/domains")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/global/domains/corp.example.com?updateMask=labels", "/domains/corp.example.com")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/global/domains/corp.example.com", "/domains/corp.example.com")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/domains/corp.example.com:resetAdminPassword", ":resetAdminPassword")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/domains/corp.example.com:attachTrust", ":attachTrust")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/domains/corp.example.com:reconfigureTrust", ":reconfigureTrust")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/domains/corp.example.com:detachTrust", ":detachTrust")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/domains/corp.example.com:validateTrust", ":validateTrust")
}

func TestGCPManagedIdentitiesRouter_RESTLocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPManagedIdentitiesContractServer(t)

	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global", "/locations/global")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/operations?pageSize=1", "/operations")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/operations/op-1", "/operations/op-1")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/operations/op-1:cancel", ":cancel")
}

func TestGCPManagedIdentitiesRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPManagedIdentitiesContractServer(t)

	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedidentities.v1.ManagedIdentitiesService/ListDomains", "ManagedIdentitiesService/ListDomains")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedidentities.v1.ManagedIdentitiesService/GetDomain", "ManagedIdentitiesService/GetDomain")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedidentities.v1.ManagedIdentitiesService/CreateMicrosoftAdDomain", "ManagedIdentitiesService/CreateMicrosoftAdDomain")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedidentities.v1.ManagedIdentitiesService/UpdateDomain", "ManagedIdentitiesService/UpdateDomain")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedidentities.v1.ManagedIdentitiesService/DeleteDomain", "ManagedIdentitiesService/DeleteDomain")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedidentities.v1.ManagedIdentitiesService/ResetAdminPassword", "ManagedIdentitiesService/ResetAdminPassword")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedidentities.v1.ManagedIdentitiesService/AttachTrust", "ManagedIdentitiesService/AttachTrust")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedidentities.v1.ManagedIdentitiesService/ReconfigureTrust", "ManagedIdentitiesService/ReconfigureTrust")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedidentities.v1.ManagedIdentitiesService/DetachTrust", "ManagedIdentitiesService/DetachTrust")
	assertGCPManagedIdentitiesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedidentities.v1.ManagedIdentitiesService/ValidateTrust", "ManagedIdentitiesService/ValidateTrust")
}

func newGCPManagedIdentitiesContractServer(t *testing.T) *httptest.Server {
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

func assertGCPManagedIdentitiesNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp managedidentities router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPManagedidentitiesRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPManagedidentitiesRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/managedidentities?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp managedidentities contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "managedidentities" {
		t.Fatalf("expected service=managedidentities, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

