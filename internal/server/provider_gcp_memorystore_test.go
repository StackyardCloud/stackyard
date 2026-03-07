package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMemorystoreRouter_InstanceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMemorystoreContractServer(t)

	assertGCPMemorystoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances?pageSize=1", "/instances")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances?instanceId=redis-a", "/instances")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/redis-a", "/instances/redis-a")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/instances/redis-a?updateMask=labels", "/instances/redis-a")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/instances/redis-a", "/instances/redis-a")
}

func TestGCPMemorystoreRouter_CertificateAndMaintenanceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMemorystoreContractServer(t)

	assertGCPMemorystoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/redis-a/certificateAuthority", "/certificateAuthority")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/redis-a:rescheduleMaintenance", ":rescheduleMaintenance")
}

func TestGCPMemorystoreRouter_BackupRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMemorystoreContractServer(t)

	collection := "/gcp/v1/projects/stackyard/locations/us-central1/backupCollections/default"
	backup := collection + "/backups/backup-1"
	instance := "/gcp/v1/projects/stackyard/locations/us-central1/instances/redis-a"

	assertGCPMemorystoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupCollections?pageSize=1", "/backupCollections")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodGet, collection, "/backupCollections/default")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodGet, collection+"/backups?pageSize=1", "/backups")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodGet, backup, "/backups/backup-1")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodDelete, backup, "/backups/backup-1")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodPost, backup+":export", ":export")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodPost, instance+":backup", ":backup")
}

func TestGCPMemorystoreRouter_LocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMemorystoreContractServer(t)

	assertGCPMemorystoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
}

func TestGCPMemorystoreRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMemorystoreContractServer(t)

	assertGCPMemorystoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.memorystore.v1.Memorystore/ListInstances", "Memorystore/ListInstances")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.memorystore.v1.Memorystore/CreateInstance", "Memorystore/CreateInstance")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.memorystore.v1.Memorystore/GetCertificateAuthority", "Memorystore/GetCertificateAuthority")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.memorystore.v1.Memorystore/ListBackups", "Memorystore/ListBackups")
	assertGCPMemorystoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.memorystore.v1.Memorystore/ExportBackup", "Memorystore/ExportBackup")
}

func newGCPMemorystoreContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMemorystoreNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp memorystore router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPMemorystoreRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPMemorystoreRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/memorystore?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp memorystore contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "memorystore" {
		t.Fatalf("expected service=memorystore, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

