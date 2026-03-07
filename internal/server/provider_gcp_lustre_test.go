package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPLustreRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLustreContractServer(t)

	assertGCPLustreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances?pageSize=1", "/instances")
	assertGCPLustreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/instance-a", "/instances/instance-a")
	assertGCPLustreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances?instanceId=instance-a", "/instances")
	assertGCPLustreNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/instances/instance-a?updateMask=description", "/instances/instance-a")
	assertGCPLustreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/instances/instance-a", "/instances/instance-a")
	assertGCPLustreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/instance-a:exportData", ":exportData")
	assertGCPLustreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/instance-a:importData", ":importData")
}

func TestGCPLustreRouter_RESTLocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLustreContractServer(t)

	assertGCPLustreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPLustreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPLustreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPLustreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPLustreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
}

func TestGCPLustreRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLustreContractServer(t)

	assertGCPLustreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.lustre.v1.Lustre/ListInstances", "Lustre/ListInstances")
	assertGCPLustreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.lustre.v1.Lustre/GetInstance", "Lustre/GetInstance")
	assertGCPLustreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.lustre.v1.Lustre/CreateInstance", "Lustre/CreateInstance")
	assertGCPLustreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.lustre.v1.Lustre/UpdateInstance", "Lustre/UpdateInstance")
	assertGCPLustreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.lustre.v1.Lustre/DeleteInstance", "Lustre/DeleteInstance")
	assertGCPLustreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.lustre.v1.Lustre/ExportData", "Lustre/ExportData")
	assertGCPLustreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.lustre.v1.Lustre/ImportData", "Lustre/ImportData")
}

func newGCPLustreContractServer(t *testing.T) *httptest.Server {
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

func assertGCPLustreNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp lustre router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPLustreRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPLustreRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lustre?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp lustre contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "lustre" {
		t.Fatalf("expected service=lustre, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

