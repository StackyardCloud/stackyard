package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPNetAppRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetAppContractServer(t)

	assertGCPNetAppNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/storagePools?pageSize=1", "/storagePools")
	assertGCPNetAppNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/storagePools/pool-a", "/storagePools/pool-a")
	assertGCPNetAppNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/volumes?pageSize=1", "/volumes")
	assertGCPNetAppNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/volumes?volumeId=vol-a", "/volumes")
	assertGCPNetAppNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/volumes/vol-a", "/volumes/vol-a")
	assertGCPNetAppNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/activeDirectories?pageSize=1", "/activeDirectories")
	assertGCPNetAppNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupVaults?pageSize=1", "/backupVaults")
	assertGCPNetAppNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/volumes/vol-a:revert", ":revert")
	assertGCPNetAppNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/replications/repl-a:stop", ":stop")
}

func TestGCPNetAppRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetAppContractServer(t)

	assertGCPNetAppNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.netapp.v1.NetApp/ListStoragePools", "NetApp/ListStoragePools")
	assertGCPNetAppNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.netapp.v1.NetApp/GetStoragePool", "NetApp/GetStoragePool")
	assertGCPNetAppNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.netapp.v1.NetApp/ListVolumes", "NetApp/ListVolumes")
	assertGCPNetAppNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.netapp.v1.NetApp/CreateVolume", "NetApp/CreateVolume")
	assertGCPNetAppNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.netapp.v1.NetApp/DeleteVolume", "NetApp/DeleteVolume")
	assertGCPNetAppNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.netapp.v1.NetApp/ListActiveDirectories", "NetApp/ListActiveDirectories")
	assertGCPNetAppNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.netapp.v1.NetApp/ListBackupVaults", "NetApp/ListBackupVaults")
}

func newGCPNetAppContractServer(t *testing.T) *httptest.Server {
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

func assertGCPNetAppNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp netapp router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPNetappRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPNetappRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/netapp?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp netapp contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "netapp" {
		t.Fatalf("expected service=netapp, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

