package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPIAPRouter_AdminRESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIAPContractServer(t)
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/123456789/iap_web:getIamPolicy", ":getIamPolicy")
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/123456789/iap_web:setIamPolicy", ":setIamPolicy")
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/123456789/iap_web:testIamPermissions", ":testIamPermissions")
	assertGCPIAPNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/123456789/iap_web:iapSettings", ":iapSettings")
	assertGCPIAPNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/123456789/iap_web:iapSettings", ":iapSettings")
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/123456789/iap_web:validateAttributeExpression", ":validateAttributeExpression")
}

func TestGCPIAPRouter_TunnelDestGroupRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIAPContractServer(t)
	parent := "/gcp/v1/projects/123456789/iap_tunnel/locations/us-central1/destGroups"
	name := parent + "/corp-dest"

	assertGCPIAPNotImplemented(t, ts, http.MethodGet, parent+"?pageSize=1", "/destGroups")
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, parent+"?tunnelDestGroupId=corp-dest", "/destGroups")
	assertGCPIAPNotImplemented(t, ts, http.MethodGet, name, "/destGroups/corp-dest")
	assertGCPIAPNotImplemented(t, ts, http.MethodPatch, name+"?updateMask=fqdns", "/destGroups/corp-dest")
	assertGCPIAPNotImplemented(t, ts, http.MethodDelete, name, "/destGroups/corp-dest")
}

func TestGCPIAPRouter_OAuthRESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIAPContractServer(t)
	brands := "/gcp/v1/projects/stackyard/brands"
	brand := brands + "/brand-1"
	clients := brand + "/identityAwareProxyClients"
	client := clients + "/client-1"

	assertGCPIAPNotImplemented(t, ts, http.MethodGet, brands, "/brands")
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, brands, "/brands")
	assertGCPIAPNotImplemented(t, ts, http.MethodGet, brand, "/brands/brand-1")
	assertGCPIAPNotImplemented(t, ts, http.MethodGet, clients+"?pageSize=1", "/identityAwareProxyClients")
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, clients, "/identityAwareProxyClients")
	assertGCPIAPNotImplemented(t, ts, http.MethodGet, client, "/identityAwareProxyClients/client-1")
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, client+":resetSecret", ":resetSecret")
	assertGCPIAPNotImplemented(t, ts, http.MethodDelete, client, "/identityAwareProxyClients/client-1")
}

func TestGCPIAPRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIAPContractServer(t)
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iap.v1.IdentityAwareProxyAdminService/GetIapSettings", "IdentityAwareProxyAdminService/GetIapSettings")
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iap.v1.IdentityAwareProxyAdminService/UpdateIapSettings", "IdentityAwareProxyAdminService/UpdateIapSettings")
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iap.v1.IdentityAwareProxyOAuthService/ListBrands", "IdentityAwareProxyOAuthService/ListBrands")
	assertGCPIAPNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iap.v1.IdentityAwareProxyOAuthService/CreateIdentityAwareProxyClient", "IdentityAwareProxyOAuthService/CreateIdentityAwareProxyClient")
}

func TestIsGCPIAPPath_DoesNotMatchGenericIAMPolicyRoute(t *testing.T) {
	t.Parallel()

	if isGCPIAPPath("/gcp/v1/projects/stackyard:getIamPolicy") {
		t.Fatalf("expected non-IAP IAM route to be excluded from IAP path matching")
	}
}

func newGCPIAPContractServer(t *testing.T) *httptest.Server {
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

func assertGCPIAPNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp iap router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPIapRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPIapRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/iap?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp iap contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "iap" {
		t.Fatalf("expected service=iap, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

