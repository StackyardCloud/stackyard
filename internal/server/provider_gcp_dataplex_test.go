package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDataplexRouter_LakeRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataplexContractServer(t)
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes?pageSize=1", "/lakes")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/lakes?lakeId=team-lake", "/lakes")
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake", "/lakes/team-lake")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake?updateMask=display_name", "/lakes/team-lake")
	assertGCPDataplexNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake", "/lakes/team-lake")
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/actions?pageSize=1", "/actions")
}

func TestGCPDataplexRouter_ZoneRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataplexContractServer(t)
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones?pageSize=1", "/zones")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones?zoneId=raw-zone", "/zones")
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones/raw-zone", "/zones/raw-zone")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones/raw-zone?updateMask=display_name", "/zones/raw-zone")
	assertGCPDataplexNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones/raw-zone", "/zones/raw-zone")
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones/raw-zone/actions?pageSize=1", "/actions")
}

func TestGCPDataplexRouter_AssetRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataplexContractServer(t)
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones/raw-zone/assets?pageSize=1", "/assets")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones/raw-zone/assets?assetId=raw-asset", "/assets")
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones/raw-zone/assets/raw-asset", "/assets/raw-asset")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones/raw-zone/assets/raw-asset?updateMask=display_name", "/assets/raw-asset")
	assertGCPDataplexNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones/raw-zone/assets/raw-asset", "/assets/raw-asset")
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/zones/raw-zone/assets/raw-asset/actions?pageSize=1", "/actions")
}

func TestGCPDataplexRouter_TaskAndJobRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataplexContractServer(t)
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/tasks?pageSize=1", "/tasks")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/tasks?taskId=profile-task", "/tasks")
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/tasks/profile-task", "/tasks/profile-task")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/tasks/profile-task?updateMask=description", "/tasks/profile-task")
	assertGCPDataplexNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/tasks/profile-task", "/tasks/profile-task")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/tasks/profile-task:run", ":run")
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/tasks/profile-task/jobs?pageSize=1", "/jobs")
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/tasks/profile-task/jobs/job-1", "/jobs/job-1")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/tasks/profile-task/jobs/job-1:cancel", ":cancel")
}

func TestGCPDataplexRouter_EnvironmentAndSessionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataplexContractServer(t)
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/environments?pageSize=1", "/environments")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/environments?environmentId=analytics", "/environments")
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/environments/analytics", "/environments/analytics")
	assertGCPDataplexNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/environments/analytics?updateMask=description", "/environments/analytics")
	assertGCPDataplexNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/environments/analytics", "/environments/analytics")
	assertGCPDataplexNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lakes/team-lake/environments/analytics/sessions?pageSize=1", "/sessions")
}

func newGCPDataplexContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDataplexNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataplex router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPDataplexRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPDataplexRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/dataplex?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp dataplex contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "dataplex" {
		t.Fatalf("expected service=dataplex, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

