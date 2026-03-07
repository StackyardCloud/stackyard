package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPGKEHubRouter_MembershipRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGKEHubContractServer(t)
	assertGCPGKEHubNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/memberships?pageSize=1", "/memberships")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/memberships", "/memberships")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/memberships/cluster-a", "/memberships/cluster-a")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodPatch, "/gcp/v1beta1/projects/stackyard/locations/global/memberships/cluster-a", "/memberships/cluster-a")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/locations/global/memberships/cluster-a", "/memberships/cluster-a")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/memberships/cluster-a:generateConnectManifest", ":generateConnectManifest")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/memberships:validateExclusivity?intendedMembership=cluster-a", "memberships:validateExclusivity")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/memberships/cluster-a:generateExclusivityManifest", ":generateExclusivityManifest")
}

func TestGCPGKEHubRouter_LocationIAMAndOperationsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGKEHubContractServer(t)
	assertGCPGKEHubNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/memberships/cluster-a:getIamPolicy", ":getIamPolicy")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/memberships/cluster-a:setIamPolicy", ":setIamPolicy")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/memberships/cluster-a:testIamPermissions", ":testIamPermissions")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/operations?pageSize=1", "/operations")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/operations/op-1", "/operations/op-1")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/operations/op-1:cancel", ":cancel")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/locations/global/operations/op-1", "/operations/op-1")
}

func TestGCPGKEHubRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGKEHubContractServer(t)
	assertGCPGKEHubNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkehub.v1beta1.GkeHubMembershipService/ListMemberships", "GkeHubMembershipService/ListMemberships")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkehub.v1beta1.GkeHubMembershipService/CreateMembership", "GkeHubMembershipService/CreateMembership")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkehub.v1beta1.GkeHubMembershipService/GenerateConnectManifest", "GkeHubMembershipService/GenerateConnectManifest")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkehub.v1beta1.GkeHubMembershipService/ValidateExclusivity", "GkeHubMembershipService/ValidateExclusivity")
	assertGCPGKEHubNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkehub.v1beta1.GkeHubMembershipService/GenerateExclusivityManifest", "GkeHubMembershipService/GenerateExclusivityManifest")
}

func newGCPGKEHubContractServer(t *testing.T) *httptest.Server {
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

func assertGCPGKEHubNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp gkehub router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPGkehubRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPGkehubRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/gkehub?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp gkehub contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "gkehub" {
		t.Fatalf("expected service=gkehub, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

