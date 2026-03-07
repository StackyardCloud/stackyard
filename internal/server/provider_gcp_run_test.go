package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPRunRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPRunContractServer(t)

	parent := "/gcp/v2/projects/stackyard/locations/us-central1"
	serviceName := parent + "/services/service-1"
	jobName := parent + "/jobs/job-1"
	executionName := jobName + "/executions/execution-1"
	taskName := executionName + "/tasks/task-1"
	revisionName := serviceName + "/revisions/service-1-00001"

	assertGCPRunSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPRunSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPRunSuccess(t, ts, http.MethodGet, parent+"/services?pageSize=1", nil, "services")
	assertGCPRunSuccess(t, ts, http.MethodGet, serviceName, nil, "service-1")
	assertGCPRunSuccess(t, ts, http.MethodPost, parent+"/services?serviceId=service-1", []byte(`{"template":{"containers":[{"image":"us-docker.pkg.dev/cloudrun/container/hello"}]}}`), "operations/createService.service-1")
	assertGCPRunSuccess(t, ts, http.MethodPatch, serviceName+"?updateMask=template.containers", []byte(`{"name":"projects/stackyard/locations/us-central1/services/service-1","template":{"containers":[{"image":"us-docker.pkg.dev/cloudrun/container/hello"}]}}`), "operations/updateService.service-1")
	assertGCPRunSuccess(t, ts, http.MethodDelete, serviceName, nil, "operations/deleteService.service-1")

	assertGCPRunSuccess(t, ts, http.MethodGet, parent+"/jobs?pageSize=1", nil, "jobs")
	assertGCPRunSuccess(t, ts, http.MethodGet, jobName, nil, "job-1")
	assertGCPRunSuccess(t, ts, http.MethodPost, parent+"/jobs?jobId=job-1", []byte(`{"template":{"template":{"containers":[{"image":"us-docker.pkg.dev/cloudrun/container/job"}]}}}`), "operations/createJob.job-1")
	assertGCPRunSuccess(t, ts, http.MethodPatch, jobName, []byte(`{"name":"projects/stackyard/locations/us-central1/jobs/job-1","template":{"template":{"containers":[{"image":"us-docker.pkg.dev/cloudrun/container/job"}]}}}`), "operations/updateJob.job-1")
	assertGCPRunSuccess(t, ts, http.MethodPost, jobName+":run", []byte(`{"name":"projects/stackyard/locations/us-central1/jobs/job-1"}`), "operations/runJob.job-1")
	assertGCPRunSuccess(t, ts, http.MethodDelete, jobName, nil, "operations/deleteJob.job-1")

	assertGCPRunSuccess(t, ts, http.MethodGet, jobName+"/executions?pageSize=1", nil, "executions")
	assertGCPRunSuccess(t, ts, http.MethodGet, executionName, nil, "execution-1")
	assertGCPRunSuccess(t, ts, http.MethodPost, executionName+":cancel", []byte(`{"name":"projects/stackyard/locations/us-central1/jobs/job-1/executions/execution-1"}`), "operations/cancelExecution.execution-1")
	assertGCPRunSuccess(t, ts, http.MethodDelete, executionName, nil, "operations/deleteExecution.execution-1")

	assertGCPRunSuccess(t, ts, http.MethodGet, executionName+"/tasks?pageSize=1", nil, "tasks")
	assertGCPRunSuccess(t, ts, http.MethodGet, taskName, nil, "task-1")

	assertGCPRunSuccess(t, ts, http.MethodGet, serviceName+"/revisions?pageSize=1", nil, "revisions")
	assertGCPRunSuccess(t, ts, http.MethodGet, revisionName, nil, "service-1-00001")
	assertGCPRunSuccess(t, ts, http.MethodDelete, revisionName, nil, "operations/deleteRevision.service-1-00001")

	assertGCPRunSuccess(t, ts, http.MethodGet, parent+"/operations?pageSize=1", nil, "operations")
	assertGCPRunSuccess(t, ts, http.MethodGet, parent+"/operations/createService.service-1", nil, "createService.service-1")
	assertGCPRunSuccess(t, ts, http.MethodPost, parent+"/operations/createService.service-1:wait", []byte(`{}`), "createService.service-1")
}

func TestGCPRunRouter_ListServicesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPRunContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/services?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "run",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp run router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRunRouter_CreateServiceRequiresServiceID(t *testing.T) {
	t.Parallel()

	ts := newGCPRunContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/services", []byte(`{"template":{"containers":[{"image":"us-docker.pkg.dev/cloudrun/container/hello"}]}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "run",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp run router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRunRouter_UpdateServiceNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPRunContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v2/projects/stackyard/locations/us-central1/services/service-1?updateMask=template.containers", []byte(`{"name":"projects/stackyard/locations/us-central1/services/service-2","template":{"containers":[{"image":"us-docker.pkg.dev/cloudrun/container/hello"}]}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "run",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp run router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRunRouter_CreateJobRequiresTemplate(t *testing.T) {
	t.Parallel()

	ts := newGCPRunContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/jobs?jobId=job-1", []byte(`{"labels":{"env":"test"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "run",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp run router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRunRouter_RunJobNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPRunContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/jobs/job-1:run", []byte(`{"name":"projects/stackyard/locations/us-central1/jobs/job-2"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "run",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp run router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRunRouter_DeleteExecutionEtagMismatch(t *testing.T) {
	t.Parallel()

	ts := newGCPRunContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/locations/us-central1/jobs/job-1/executions/execution-1?etag=stale", nil, map[string]string{
		"X-Stackyard-GCP-Service": "run",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp run router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRunRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/run?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp run contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "run" {
		t.Fatalf("expected service=run, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPRunContractServer(t *testing.T) *httptest.Server {
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

func assertGCPRunSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "run",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp run router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
