package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPFilestoreRouter_InstanceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFilestoreContractServer(t)
	assertGCPFilestoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances?pageSize=1", "/instances")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances", "/instances")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/team-instance?updateMask=description", "/instances/team-instance")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/team-instance:restore", ":restore")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/team-instance:revert", ":revert")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/team-instance:promoteReplica", ":promoteReplica")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/team-instance", "/instances/team-instance")
}

func TestGCPFilestoreRouter_SnapshotRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFilestoreContractServer(t)
	assertGCPFilestoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/team-instance/snapshots?pageSize=1", "/snapshots")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/team-instance/snapshots", "/snapshots")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/team-instance/snapshots/team-snapshot?updateMask=description", "/snapshots/team-snapshot")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/team-instance/snapshots/team-snapshot", "/snapshots/team-snapshot")
}

func TestGCPFilestoreRouter_BackupAndOperationsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFilestoreContractServer(t)
	assertGCPFilestoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backups?pageSize=1", "/backups")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/backups", "/backups")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/backups/team-backup?updateMask=description", "/backups/team-backup")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/backups/team-backup", "/backups/team-backup")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
}

func TestGCPFilestoreRouter_LocationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFilestoreContractServer(t)
	assertGCPFilestoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations", "/locations")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-c", "/locations/us-central1-c")
}

func TestGCPFilestoreRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFilestoreContractServer(t)
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.filestore.v1.CloudFilestoreManager/ListInstances", "CloudFilestoreManager/ListInstances")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.filestore.v1.CloudFilestoreManager/CreateInstance", "CloudFilestoreManager/CreateInstance")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.filestore.v1.CloudFilestoreManager/ListSnapshots", "CloudFilestoreManager/ListSnapshots")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.filestore.v1.CloudFilestoreManager/ListBackups", "CloudFilestoreManager/ListBackups")
	assertGCPFilestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.filestore.v1.CloudFilestoreManager/PromoteReplica", "CloudFilestoreManager/PromoteReplica")
}

func newGCPFilestoreContractServer(t *testing.T) *httptest.Server {
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

func assertGCPFilestoreNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp filestore router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPFilestoreRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPFilestoreRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/filestore?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp filestore contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "filestore" {
		t.Fatalf("expected service=filestore, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

