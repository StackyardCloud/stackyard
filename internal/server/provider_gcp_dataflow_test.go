package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDataflowRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataflowContractServer(t)

	assertGCPDataflowSuccess(t, ts, http.MethodPost, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs", []byte(`{"name":"team-job"}`), "JOB_STATE_RUNNING")
	assertGCPDataflowSuccess(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs/team-job", nil, "team-job")
	assertGCPDataflowSuccess(t, ts, http.MethodPut, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs/team-job", []byte(`{"requestedState":"JOB_STATE_CANCELLED"}`), "team-job")
	assertGCPDataflowSuccess(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs?pageSize=1", nil, "jobs")
	assertGCPDataflowSuccess(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/jobs:aggregated?pageSize=1", nil, "jobs")
	assertGCPDataflowSuccess(t, ts, http.MethodPost, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs/team-job:snapshot", []byte(`{}`), "snap-1")
	assertGCPDataflowSuccess(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs/team-job/messages?pageSize=1", nil, "jobMessages")
	assertGCPDataflowSuccess(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs/team-job/metrics", nil, "metrics")
	assertGCPDataflowSuccess(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs/team-job/executionDetails", nil, "{}")
	assertGCPDataflowSuccess(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs/team-job/stages/stage-a/executionDetails", nil, "{}")
	assertGCPDataflowSuccess(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/locations/us-central1/snapshots/snap-1", nil, "snap-1")
	assertGCPDataflowSuccess(t, ts, http.MethodDelete, "/gcp/v1b3/projects/stackyard/locations/us-central1/snapshots/snap-1", nil, "{}")
	assertGCPDataflowSuccess(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs/team-job/snapshots", nil, "snapshots")
	assertGCPDataflowSuccess(t, ts, http.MethodPost, "/gcp/v1b3/projects/stackyard/locations/us-central1/templates", []byte(`{"jobName":"template-job","gcsPath":"gs://stackyard/templates/wordcount"}`), "template-job")
	assertGCPDataflowSuccess(t, ts, http.MethodPost, "/gcp/v1b3/projects/stackyard/locations/us-central1/templates:launch", []byte(`{"jobName":"launch-template-job"}`), "launch-template-job")
	assertGCPDataflowSuccess(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/locations/us-central1/templates:get?gcsPath=gs://stackyard/templates/wordcount", nil, "{}")
	assertGCPDataflowSuccess(t, ts, http.MethodPost, "/gcp/v1b3/projects/stackyard/locations/us-central1/flexTemplates:launch", []byte(`{"launchParameter":{"jobName":"flex-template-job","containerSpecGcsPath":"gs://stackyard/flex/containerSpec.json"}}`), "flex-template-job")
}

func TestGCPDataflowRouter_ListJobsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPDataflowContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp dataflow router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPDataflowRouter_CreateJobRequiresName(t *testing.T) {
	t.Parallel()

	ts := newGCPDataflowContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp dataflow router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPDataflowRouter_LaunchTemplateRequiresJobName(t *testing.T) {
	t.Parallel()

	ts := newGCPDataflowContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1b3/projects/stackyard/locations/us-central1/templates:launch", []byte(`{"launchParameters":{}}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp dataflow router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPDataflowRouter_ListJobMessagesPageTokenOutOfRange(t *testing.T) {
	t.Parallel()

	ts := newGCPDataflowContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1b3/projects/stackyard/locations/us-central1/jobs/team-job/messages?pageToken=99", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp dataflow router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPDataflowContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDataflowSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp dataflow router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
