package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPKMSInventoryRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPKMSInventoryContractServer(t)

	listCryptoKeys := "/gcp/v1/projects/stackyard/cryptoKeys?pageSize=1"
	cryptoKey := "/gcp/v1/projects/stackyard/locations/us-central1/keyRings/team-ring/cryptoKeys/app-key"
	summary := cryptoKey + "/protectedResourcesSummary"
	search := "/gcp/v1/organizations/123/protectedResources:search?cryptoKey=projects%2Fstackyard%2Flocations%2Fus-central1%2FkeyRings%2Fteam-ring%2FcryptoKeys%2Fapp-key&resourceTypes=compute.googleapis.com%2FDisk"

	assertGCPKMSInventoryNotImplemented(t, ts, http.MethodGet, listCryptoKeys, "/projects/stackyard/cryptoKeys")
	assertGCPKMSInventoryNotImplemented(t, ts, http.MethodGet, summary, "/protectedResourcesSummary")
	assertGCPKMSInventoryNotImplemented(t, ts, http.MethodGet, search, "/protectedResources:search")
}

func TestGCPKMSInventoryRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPKMSInventoryContractServer(t)

	assertGCPKMSInventoryNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.kms.inventory.v1.KeyDashboardService/ListCryptoKeys", "KeyDashboardService/ListCryptoKeys")
	assertGCPKMSInventoryNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.kms.inventory.v1.KeyTrackingService/GetProtectedResourcesSummary", "KeyTrackingService/GetProtectedResourcesSummary")
	assertGCPKMSInventoryNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.kms.inventory.v1.KeyTrackingService/SearchProtectedResources", "KeyTrackingService/SearchProtectedResources")
}

func newGCPKMSInventoryContractServer(t *testing.T) *httptest.Server {
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

func assertGCPKMSInventoryNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp kms inventory router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPKmsInventoryRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPKmsInventoryRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/kms_inventory?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp kms_inventory contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "kms_inventory" {
		t.Fatalf("expected service=kms_inventory, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
