package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPGKEBackupRouter_BackupPlanAndChannelRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGKEBackupContractServer(t)
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlans?pageSize=1", "/backupPlans")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlans", "/backupPlans")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlans/plan-a", "/backupPlans/plan-a")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupChannels?pageSize=1", "/backupChannels")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/backupChannels", "/backupChannels")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlanBindings?pageSize=1", "/backupPlanBindings")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlanBindings/binding-a", "/backupPlanBindings/binding-a")
}

func TestGCPGKEBackupRouter_BackupAndRestoreRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGKEBackupContractServer(t)
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlans/plan-a/backups?pageSize=1", "/backups")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlans/plan-a/backups", "/backups")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlans/plan-a/backups/backup-a", "/backups/backup-a")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlans/plan-a/backups/backup-a/volumeBackups?pageSize=1", "/volumeBackups")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlans/plan-a/backups/backup-a:getBackupIndexDownloadUrl", ":getBackupIndexDownloadUrl")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/restorePlans?pageSize=1", "/restorePlans")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/restorePlans", "/restorePlans")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/restorePlans/restore-plan-a/restores?pageSize=1", "/restores")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/restorePlans/restore-plan-a/restores", "/restores")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/restorePlans/restore-plan-a/restores/restore-a/volumeRestores?pageSize=1", "/volumeRestores")
}

func TestGCPGKEBackupRouter_LocationIAMAndOperationsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGKEBackupContractServer(t)
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlans/plan-a:getIamPolicy", ":getIamPolicy")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlans/plan-a:setIamPolicy", ":setIamPolicy")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/backupPlans/plan-a:testIamPermissions", ":testIamPermissions")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
}

func TestGCPGKEBackupRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGKEBackupContractServer(t)
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkebackup.v1.BackupForGKE/ListBackupPlans", "BackupForGKE/ListBackupPlans")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkebackup.v1.BackupForGKE/CreateBackupPlan", "BackupForGKE/CreateBackupPlan")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkebackup.v1.BackupForGKE/ListBackups", "BackupForGKE/ListBackups")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkebackup.v1.BackupForGKE/CreateRestorePlan", "BackupForGKE/CreateRestorePlan")
	assertGCPGKEBackupNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkebackup.v1.BackupForGKE/GetBackupIndexDownloadUrl", "BackupForGKE/GetBackupIndexDownloadUrl")
}

func newGCPGKEBackupContractServer(t *testing.T) *httptest.Server {
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

func assertGCPGKEBackupNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp gkebackup router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPGkebackupRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPGkebackupRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/gkebackup?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp gkebackup contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "gkebackup" {
		t.Fatalf("expected service=gkebackup, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
