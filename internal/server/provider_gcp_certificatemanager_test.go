package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCertificateManagerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCertificateManagerContractServer(t)
	base := "/gcp/v1/projects/stackyard/locations/global"

	assertGCPCertificateManagerSuccess(t, ts, http.MethodGet, base+"/certificates?pageSize=1", nil, "certificates")
	assertGCPCertificateManagerSuccess(t, ts, http.MethodGet, base+"/certificates/team-certificate", nil, "team-certificate")
	assertGCPCertificateManagerSuccess(t, ts, http.MethodGet, base+"/certificateMaps?pageSize=1", nil, "certificateMaps")
	assertGCPCertificateManagerSuccess(t, ts, http.MethodGet, base+"/certificateMaps/team-map/certificateMapEntries?pageSize=1", nil, "certificateMapEntries")
	assertGCPCertificateManagerSuccess(t, ts, http.MethodGet, base+"/dnsAuthorizations?pageSize=1", nil, "dnsAuthorizations")
	assertGCPCertificateManagerSuccess(t, ts, http.MethodGet, base+"/trustConfigs?pageSize=1", nil, "trustConfigs")
	assertGCPCertificateManagerSuccess(t, ts, http.MethodGet, base+"/certificateIssuanceConfigs?pageSize=1", nil, "certificateIssuanceConfigs")
	assertGCPCertificateManagerSuccess(t, ts, http.MethodPost, base+"/certificates?certificateId=team-certificate", []byte(`{"certificate":{"description":"test"}}`), "operations")
	assertGCPCertificateManagerSuccess(t, ts, http.MethodPost, base+"/dnsAuthorizations?dnsAuthorizationId=team-dns-auth", []byte(`{"dnsAuthorization":{"domain":"example.com"}}`), "operations")
	assertGCPCertificateManagerSuccess(t, ts, http.MethodDelete, base+"/trustConfigs/team-trust", nil, "operations")
}

func TestGCPCertificateManagerRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPCertificateManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.certificatemanager.v1.CertificateManager/ListCertificates", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp certificatemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, "CertificateManager/ListCertificates") {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCertificateManagerRouter_ListCertificatesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPCertificateManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/certificates?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp certificatemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCertificateManagerRouter_CreateCertificateRequiresBody(t *testing.T) {
	t.Parallel()

	ts := newGCPCertificateManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/certificates?certificateId=team-certificate", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp certificatemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPCertificateManagerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPCertificateManagerSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp certificatemanager router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
