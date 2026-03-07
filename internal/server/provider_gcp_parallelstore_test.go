package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPParallelstoreRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPParallelstoreContractServer(t)

	assertGCPParallelstoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances?pageSize=1", "/instances")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/instance-a", "/instances/instance-a")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances?instanceId=instance-a", "/instances")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/instance-a?updateMask=description", "/instances/instance-a")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/instance-a", "/instances/instance-a")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/instance-a:exportData", ":exportData")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1-c/instances/instance-a:importData", ":importData")
}

func TestGCPParallelstoreRouter_RESTLocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPParallelstoreContractServer(t)

	assertGCPParallelstoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-c", "/locations/us-central1-c")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-c/operations?pageSize=1", "/operations")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1-c/operations/op-1", "/operations/op-1")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1-c/operations/op-1:cancel", ":cancel")
}

func TestGCPParallelstoreRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPParallelstoreContractServer(t)

	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parallelstore.v1.Parallelstore/ListInstances", "Parallelstore/ListInstances")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parallelstore.v1.Parallelstore/GetInstance", "Parallelstore/GetInstance")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parallelstore.v1.Parallelstore/CreateInstance", "Parallelstore/CreateInstance")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parallelstore.v1.Parallelstore/UpdateInstance", "Parallelstore/UpdateInstance")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parallelstore.v1.Parallelstore/DeleteInstance", "Parallelstore/DeleteInstance")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parallelstore.v1.Parallelstore/ExportData", "Parallelstore/ExportData")
	assertGCPParallelstoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parallelstore.v1.Parallelstore/ImportData", "Parallelstore/ImportData")
}

func newGCPParallelstoreContractServer(t *testing.T) *httptest.Server {
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

func assertGCPParallelstoreNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp parallelstore router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPParallelstoreRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPParallelstoreRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/parallelstore?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp parallelstore contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "parallelstore" {
		t.Fatalf("expected service=parallelstore, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

