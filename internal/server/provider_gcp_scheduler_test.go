package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSchedulerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSchedulerContractServer(t)

	parent := "/gcp/v1/projects/stackyard/locations/us-central1"
	jobName := parent + "/jobs/job-1"

	assertGCPSchedulerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPSchedulerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPSchedulerSuccess(t, ts, http.MethodGet, parent+"/jobs?pageSize=1", nil, "jobs")
	assertGCPSchedulerSuccess(t, ts, http.MethodGet, jobName, nil, "jobs/job-1")
	assertGCPSchedulerSuccess(t, ts, http.MethodPost, parent+"/jobs?jobId=job-1", []byte(`{"description":"team job","schedule":"*/10 * * * *","timeZone":"UTC","httpTarget":{"uri":"https://example.com/hook","httpMethod":"POST"}}`), "jobs/job-1")
	assertGCPSchedulerSuccess(t, ts, http.MethodPatch, jobName+"?updateMask=schedule,timeZone", []byte(`{"name":"projects/stackyard/locations/us-central1/jobs/job-1","description":"updated job","schedule":"*/5 * * * *","timeZone":"UTC","httpTarget":{"uri":"https://example.com/hook","httpMethod":"POST"}}`), "jobs/job-1")
	assertGCPSchedulerSuccess(t, ts, http.MethodPost, jobName+":pause", []byte(`{}`), `"state":"PAUSED"`)
	assertGCPSchedulerSuccess(t, ts, http.MethodPost, parent+"/jobs/job-paused:resume", []byte(`{}`), `"state":"ENABLED"`)
	assertGCPSchedulerSuccess(t, ts, http.MethodPost, jobName+":run", []byte(`{}`), "lastAttemptTime")
	assertGCPSchedulerSuccess(t, ts, http.MethodDelete, jobName, nil, "{}")
}

func TestGCPSchedulerRouter_ListJobsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPSchedulerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/jobs?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "scheduler",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp scheduler router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSchedulerRouter_CreateJobRequiresSchedule(t *testing.T) {
	t.Parallel()

	ts := newGCPSchedulerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/jobs?jobId=job-1", []byte(`{"timeZone":"UTC","httpTarget":{"uri":"https://example.com/hook"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "scheduler",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp scheduler router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSchedulerRouter_CreateJobRequiresTarget(t *testing.T) {
	t.Parallel()

	ts := newGCPSchedulerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/jobs?jobId=job-1", []byte(`{"schedule":"*/15 * * * *","timeZone":"UTC"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "scheduler",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp scheduler router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSchedulerRouter_UpdateJobRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPSchedulerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/jobs/job-1", []byte(`{"name":"projects/stackyard/locations/us-central1/jobs/job-1","schedule":"*/5 * * * *","timeZone":"UTC","httpTarget":{"uri":"https://example.com/hook"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "scheduler",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp scheduler router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSchedulerRouter_UpdateJobNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPSchedulerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/jobs/job-1?updateMask=schedule,timeZone", []byte(`{"name":"projects/stackyard/locations/us-central1/jobs/job-2","schedule":"*/5 * * * *","timeZone":"UTC","httpTarget":{"uri":"https://example.com/hook"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "scheduler",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp scheduler router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSchedulerRouter_PauseJobRequiresEnabledState(t *testing.T) {
	t.Parallel()

	ts := newGCPSchedulerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/jobs/job-paused:pause", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "scheduler",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp scheduler router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSchedulerRouter_ResumeJobRequiresPausedState(t *testing.T) {
	t.Parallel()

	ts := newGCPSchedulerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/jobs/job-1:resume", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "scheduler",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp scheduler router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSchedulerRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/scheduler?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp scheduler contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "scheduler" {
		t.Fatalf("expected service=scheduler, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPSchedulerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSchedulerSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "scheduler",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp scheduler router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
