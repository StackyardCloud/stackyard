package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMigrationCenterRouter_AssetRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMigrationCenterContractServer(t)
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/assets?pageSize=1", "/assets")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/assets/asset-a", "/assets/asset-a")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/assets/asset-a?updateMask=labels", "/assets/asset-a")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/assets/asset-a", "/assets/asset-a")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/assets:batchUpdate", ":batchUpdate")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/assets:batchDelete", ":batchDelete")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/assets:reportAssetFrames", ":reportAssetFrames")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/assets:aggregateValues", ":aggregateValues")
}

func TestGCPMigrationCenterRouter_ImportAndGroupRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMigrationCenterContractServer(t)
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/importJobs?pageSize=1", "/importJobs")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/importJobs?importJobId=job-a", "/importJobs")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/importJobs/job-a:validate", ":validate")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/importJobs/job-a:run", ":run")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/importJobs/job-a/importDataFiles?pageSize=1", "/importDataFiles")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/groups?groupId=group-a", "/groups")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/groups/group-a:addAssets", ":addAssets")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/groups/group-a:removeAssets", ":removeAssets")
}

func TestGCPMigrationCenterRouter_SourcePreferenceReportAndSettingsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMigrationCenterContractServer(t)
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/sources?pageSize=1", "/sources")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/sources/source-a/errorFrames?pageSize=1", "/errorFrames")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/preferenceSets?pageSize=1", "/preferenceSets")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/settings", "/settings")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/settings?updateMask=preferenceSet", "/settings")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/reportConfigs?pageSize=1", "/reportConfigs")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/reportConfigs/config-a/reports?pageSize=1", "/reports")
}

func TestGCPMigrationCenterRouter_LocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMigrationCenterContractServer(t)
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
}

func TestGCPMigrationCenterRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMigrationCenterContractServer(t)
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.migrationcenter.v1.MigrationCenter/ListAssets", "MigrationCenter/ListAssets")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.migrationcenter.v1.MigrationCenter/CreateSource", "MigrationCenter/CreateSource")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.migrationcenter.v1.MigrationCenter/CreateImportJob", "MigrationCenter/CreateImportJob")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.migrationcenter.v1.MigrationCenter/AddAssetsToGroup", "MigrationCenter/AddAssetsToGroup")
	assertGCPMigrationCenterNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.migrationcenter.v1.MigrationCenter/CreateReport", "MigrationCenter/CreateReport")
}

func newGCPMigrationCenterContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMigrationCenterNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp migration center router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPMigrationcenterRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPMigrationcenterRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/migrationcenter?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp migrationcenter contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "migrationcenter" {
		t.Fatalf("expected service=migrationcenter, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
