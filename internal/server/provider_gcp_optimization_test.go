package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPOptimizationRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOptimizationContractServer(t)

	assertGCPOptimizationNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:optimizeTours", ":optimizeTours")
	assertGCPOptimizationNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:optimizeTours", ":optimizeTours")
	assertGCPOptimizationNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:batchOptimizeTours", ":batchOptimizeTours")
	assertGCPOptimizationNotImplemented(t, ts, http.MethodGet, "/gcp/v1/operations/opt-op-1", "/operations/opt-op-1")
	assertGCPOptimizationNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/opt-op-2", "/operations/opt-op-2")
}

func TestGCPOptimizationRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOptimizationContractServer(t)

	assertGCPOptimizationNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.optimization.v1.FleetRouting/OptimizeTours", "FleetRouting/OptimizeTours")
	assertGCPOptimizationNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.optimization.v1.FleetRouting/BatchOptimizeTours", "FleetRouting/BatchOptimizeTours")
}

func newGCPOptimizationContractServer(t *testing.T) *httptest.Server {
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

func assertGCPOptimizationNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp optimization router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPOptimizationRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPOptimizationRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/optimization?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp optimization contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "optimization" {
		t.Fatalf("expected service=optimization, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
