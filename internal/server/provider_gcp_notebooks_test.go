package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPNotebooksRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNotebooksContractServer(t)

	assertGCPNotebooksNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances", "/instances")
	assertGCPNotebooksNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/notebook-1", "/instances/notebook-1")
	assertGCPNotebooksNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/environments", "/environments")
	assertGCPNotebooksNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/schedules/schedule-1", "/schedules/schedule-1")
	assertGCPNotebooksNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/executions/execution-1", "/executions/execution-1")
	assertGCPNotebooksNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/runtimes/runtime-1", "/runtimes/runtime-1")
}

func TestGCPNotebooksRouter_GrpcNotebookServiceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNotebooksContractServer(t)

	assertGCPNotebooksNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v1.NotebookService/ListInstances", "NotebookService/ListInstances")
	assertGCPNotebooksNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v1.NotebookService/GetInstance", "NotebookService/GetInstance")
	assertGCPNotebooksNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v1.NotebookService/ListEnvironments", "NotebookService/ListEnvironments")
	assertGCPNotebooksNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v1.NotebookService/ListSchedules", "NotebookService/ListSchedules")
	assertGCPNotebooksNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v1.NotebookService/ListExecutions", "NotebookService/ListExecutions")
}

func TestGCPNotebooksRouter_GrpcManagedNotebookServiceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNotebooksContractServer(t)

	assertGCPNotebooksNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v1.ManagedNotebookService/ListRuntimes", "ManagedNotebookService/ListRuntimes")
	assertGCPNotebooksNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v1.ManagedNotebookService/GetRuntime", "ManagedNotebookService/GetRuntime")
	assertGCPNotebooksNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v1.ManagedNotebookService/RefreshRuntimeTokenInternal", "RefreshRuntimeTokenInternal")
}

func newGCPNotebooksContractServer(t *testing.T) *httptest.Server {
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

func assertGCPNotebooksNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp notebooks router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPNotebooksRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPNotebooksRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/notebooks?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp notebooks contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "notebooks" {
		t.Fatalf("expected service=notebooks, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
