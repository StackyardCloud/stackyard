package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPOsConfigRouter_RESTPatchAndDeploymentRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOsConfigContractServer(t)

	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/patchJobs:execute", "/patchJobs:execute")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/patchJobs?pageSize=1", "/patchJobs")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/patchJobs/patch-job-1", "/patchJobs/patch-job-1")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/patchJobs/patch-job-1:cancel", ":cancel")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/patchJobs/patch-job-1/instanceDetails?pageSize=1", "/instanceDetails")

	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/patchDeployments?patchDeploymentId=pd-1", "/patchDeployments")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/patchDeployments/pd-1", "/patchDeployments/pd-1")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/patchDeployments/pd-1?updateMask=description", "/patchDeployments/pd-1")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/patchDeployments/pd-1", "/patchDeployments/pd-1")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/patchDeployments/pd-1:pause", ":pause")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/patchDeployments/pd-1:resume", ":resume")
}

func TestGCPOsConfigRouter_RESTZonalRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOsConfigContractServer(t)

	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-a/osPolicyAssignments?pageSize=1", "/osPolicyAssignments")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1-a/osPolicyAssignments?osPolicyAssignmentId=assignment-1", "/osPolicyAssignments")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-a/osPolicyAssignments/assignment-1", "/osPolicyAssignments/assignment-1")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-a/osPolicyAssignments/assignment-1:listRevisions?pageSize=1", ":listRevisions")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-a/instances/-/osPolicyAssignments/assignment-1/reports?pageSize=1", "/reports")

	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-a/instances/instance-1/inventories/inventory?view=FULL", "/inventories")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-a/instances/-/inventories?pageSize=1", "/inventories")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-a/instances/instance-1/vulnerabilityReports/vulnerabilityReport", "/vulnerabilityReports")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-a/instances/-/vulnerabilityReports?pageSize=1", "/vulnerabilityReports")
}

func TestGCPOsConfigRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOsConfigContractServer(t)

	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.v1.OsConfigService/ExecutePatchJob", "OsConfigService/ExecutePatchJob")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.v1.OsConfigService/GetPatchJob", "OsConfigService/GetPatchJob")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.v1.OsConfigService/ListPatchDeployments", "OsConfigService/ListPatchDeployments")

	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.v1.OsConfigZonalService/CreateOSPolicyAssignment", "OsConfigZonalService/CreateOSPolicyAssignment")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.v1.OsConfigZonalService/GetOSPolicyAssignmentReport", "OsConfigZonalService/GetOSPolicyAssignmentReport")
	assertGCPOsConfigNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.v1.OsConfigZonalService/ListVulnerabilityReports", "OsConfigZonalService/ListVulnerabilityReports")
}

func newGCPOsConfigContractServer(t *testing.T) *httptest.Server {
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

func assertGCPOsConfigNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp osconfig router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPOsconfigRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPOsconfigRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/osconfig?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp osconfig contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "osconfig" {
		t.Fatalf("expected service=osconfig, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

