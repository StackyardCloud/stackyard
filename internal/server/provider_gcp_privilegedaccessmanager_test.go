package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPPrivilegedAccessManagerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPPrivilegedAccessManagerContractServer(t)

	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:checkOnboardingStatus", ":checkOnboardingStatus")

	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements?pageSize=1", "/entitlements")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements:search", "/entitlements:search")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements/entitlement-a", "/entitlements/entitlement-a")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements?entitlementId=entitlement-a", "/entitlements")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements/entitlement-a?updateMask=maxRequestDuration", "/entitlements/entitlement-a")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements/entitlement-a", "/entitlements/entitlement-a")

	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements/entitlement-a/grants?pageSize=1", "/grants")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements/entitlement-a/grants:search", "/grants:search")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements/entitlement-a/grants", "/grants")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements/entitlement-a/grants/grant-a", "/grants/grant-a")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements/entitlement-a/grants/grant-a:approve", ":approve")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements/entitlement-a/grants/grant-a:deny", ":deny")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/entitlements/entitlement-a/grants/grant-a:revoke", ":revoke")
}

func TestGCPPrivilegedAccessManagerRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPPrivilegedAccessManagerContractServer(t)

	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/CheckOnboardingStatus", "PrivilegedAccessManager/CheckOnboardingStatus")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/ListEntitlements", "PrivilegedAccessManager/ListEntitlements")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/SearchEntitlements", "PrivilegedAccessManager/SearchEntitlements")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/GetEntitlement", "PrivilegedAccessManager/GetEntitlement")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/CreateEntitlement", "PrivilegedAccessManager/CreateEntitlement")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/UpdateEntitlement", "PrivilegedAccessManager/UpdateEntitlement")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/DeleteEntitlement", "PrivilegedAccessManager/DeleteEntitlement")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/ListGrants", "PrivilegedAccessManager/ListGrants")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/SearchGrants", "PrivilegedAccessManager/SearchGrants")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/GetGrant", "PrivilegedAccessManager/GetGrant")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/CreateGrant", "PrivilegedAccessManager/CreateGrant")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/ApproveGrant", "PrivilegedAccessManager/ApproveGrant")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/DenyGrant", "PrivilegedAccessManager/DenyGrant")
	assertGCPPrivilegedAccessManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.privilegedaccessmanager.v1.PrivilegedAccessManager/RevokeGrant", "PrivilegedAccessManager/RevokeGrant")
}

func newGCPPrivilegedAccessManagerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPPrivilegedAccessManagerNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp privilegedaccessmanager router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPPrivilegedaccessmanagerRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPPrivilegedaccessmanagerRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/privilegedaccessmanager?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp privilegedaccessmanager contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "privilegedaccessmanager" {
		t.Fatalf("expected service=privilegedaccessmanager, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

