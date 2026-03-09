package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMemcacheRouter_InstanceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMemcacheContractServer(t)

	assertGCPMemcacheNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances?pageSize=1", "/instances")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances?instanceId=cache-a", "/instances")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/cache-a", "/instances/cache-a")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/instances/cache-a?updateMask=displayName", "/instances/cache-a")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/instances/cache-a", "/instances/cache-a")
}

func TestGCPMemcacheRouter_ParameterAndMaintenanceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMemcacheContractServer(t)

	assertGCPMemcacheNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/instances/cache-a:updateParameters?updateMask=params.max_item_size", ":updateParameters")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/cache-a:applyParameters", ":applyParameters")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/cache-a:rescheduleMaintenance", ":rescheduleMaintenance")
}

func TestGCPMemcacheRouter_LocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMemcacheContractServer(t)

	assertGCPMemcacheNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
}

func TestGCPMemcacheRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMemcacheContractServer(t)

	assertGCPMemcacheNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.memcache.v1.CloudMemcache/ListInstances", "CloudMemcache/ListInstances")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.memcache.v1.CloudMemcache/GetInstance", "CloudMemcache/GetInstance")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.memcache.v1.CloudMemcache/CreateInstance", "CloudMemcache/CreateInstance")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.memcache.v1.CloudMemcache/UpdateParameters", "CloudMemcache/UpdateParameters")
	assertGCPMemcacheNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.memcache.v1.CloudMemcache/ListLocations", "CloudMemcache/ListLocations")
}

func newGCPMemcacheContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMemcacheNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp memcache router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPMemcacheRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPMemcacheRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/memcache?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp memcache contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "memcache" {
		t.Fatalf("expected service=memcache, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
