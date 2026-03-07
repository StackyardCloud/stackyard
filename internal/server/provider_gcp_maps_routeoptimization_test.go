package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const routeOptimizationClientHeader = "gl-go/1.25.1 gapic/1.26.0 gax/2.16.0 rest/UNKNOWN pb/1.36"

func TestGCPMapsRouteOptimizationRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsRouteOptimizationContractServer(t)

	assertGCPMapsRouteOptimizationSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:optimizeTours", []byte(`{"model":{},"label":"stackyard-routeoptimization"}`), "requestLabel")
	assertGCPMapsRouteOptimizationSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:optimizeTours", []byte(`{"model":{},"label":"stackyard-routeoptimization"}`), "routes")
	assertGCPMapsRouteOptimizationSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:batchOptimizeTours", []byte(`{"modelConfigs":[{"displayName":"stackyard-routeopt-model"}]}`), "operations/routeopt")
	assertGCPMapsRouteOptimizationSuccess(t, ts, http.MethodGet, "/gcp/v1/operations/routeopt-op-1", nil, "operations/routeopt-op-1")
}

func TestGCPMapsRouteOptimizationRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsRouteOptimizationContractServer(t)

	assertGCPMapsRouteOptimizationNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.routeoptimization.v1.RouteOptimization/OptimizeTours", "RouteOptimization/OptimizeTours")
	assertGCPMapsRouteOptimizationNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.routeoptimization.v1.RouteOptimization/BatchOptimizeTours", "RouteOptimization/BatchOptimizeTours")
}

func TestGCPMapsRouteOptimizationRouter_OptimizeToursInvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsRouteOptimizationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:optimizeTours", []byte(`{"label":"stackyard-routeoptimization"`), map[string]string{
		"Content-Type":      "application/json",
		"x-goog-api-client": routeOptimizationClientHeader,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps routeoptimization router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMapsRouteOptimizationRouter_OptimizeToursRequiresModel(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsRouteOptimizationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:optimizeTours", []byte(`{"label":"stackyard-routeoptimization"}`), map[string]string{
		"Content-Type":      "application/json",
		"x-goog-api-client": routeOptimizationClientHeader,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps routeoptimization router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMapsRouteOptimizationRouter_BatchOptimizeToursRequiresModelConfigs(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsRouteOptimizationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:batchOptimizeTours", []byte(`{"label":"stackyard-routeoptimization","modelConfigs":[]}`), map[string]string{
		"Content-Type":      "application/json",
		"x-goog-api-client": routeOptimizationClientHeader,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps routeoptimization router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPMapsRouteOptimizationContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMapsRouteOptimizationNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp maps routeoptimization router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPMapsRouteOptimizationSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"x-goog-api-client": routeOptimizationClientHeader,
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp maps routeoptimization router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
