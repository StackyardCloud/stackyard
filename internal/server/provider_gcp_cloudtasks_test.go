package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCloudTasksRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudTasksContractServer(t)

	assertGCPCloudTasksSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/queues?pageSize=1", nil, "queues")
	assertGCPCloudTasksSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue", nil, "queues/team-queue")
	assertGCPCloudTasksSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/queues", []byte(`{"name":"projects/stackyard/locations/us-central1/queues/team-queue"}`), "queues/team-queue")
	assertGCPCloudTasksSuccess(t, ts, http.MethodPatch, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue", []byte(`{"name":"projects/stackyard/locations/us-central1/queues/team-queue"}`), "queues/team-queue")
	assertGCPCloudTasksSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue:pause", []byte(`{}`), "PAUSED")
	assertGCPCloudTasksSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue:resume", []byte(`{}`), "RUNNING")
	assertGCPCloudTasksSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue:purge", []byte(`{}`), "purgeTime")
	assertGCPCloudTasksSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue/tasks?pageSize=1", nil, "tasks")
	assertGCPCloudTasksSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue/tasks/task-1?responseView=BASIC", nil, "tasks/task-1")
	assertGCPCloudTasksSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue/tasks?responseView=BASIC", []byte(`{"name":"projects/stackyard/locations/us-central1/queues/team-queue/tasks/task-1"}`), "tasks/task-1")
	assertGCPCloudTasksSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue/tasks/task-1:run?responseView=BASIC", []byte(`{}`), "tasks/task-1")
	assertGCPCloudTasksSuccess(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue/tasks/task-1", nil, "{}")
	assertGCPCloudTasksSuccess(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue", nil, "{}")
}

func TestGCPCloudTasksRouter_ListQueuesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudTasksContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/queues?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudtasks router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudTasksRouter_GetTaskInvalidResponseView(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudTasksContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue/tasks/task-1?responseView=INVALID", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudtasks router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudTasksRouter_CreateTaskRequiresTaskName(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudTasksContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue/tasks", []byte(`{"httpRequest":{"url":"https://example.com"}}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudtasks router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudTasksRouter_ListTasksPageTokenOutOfRange(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudTasksContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/queues/team-queue/tasks?pageToken=99", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudtasks router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPCloudTasksContractServer(t *testing.T) *httptest.Server {
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

func assertGCPCloudTasksSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp cloudtasks router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
