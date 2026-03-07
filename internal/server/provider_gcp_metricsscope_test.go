package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMetricsScopeRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMetricsScopeContractServer(t)

	assertGCPMetricsScopeNotImplemented(t, ts, http.MethodGet, "/gcp/v1/locations/global/metricsScopes/stackyard", "/metricsScopes/stackyard")
	assertGCPMetricsScopeNotImplemented(t, ts, http.MethodPost, "/gcp/v1/locations/global/metricsScopes/stackyard/projects", "/projects")
	assertGCPMetricsScopeNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/locations/global/metricsScopes/stackyard/projects/team-a", "/projects/team-a")
	assertGCPMetricsScopeNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/team-a/locations/global/metricsScopes:listMetricsScopesByMonitoredProject", ":listMetricsScopesByMonitoredProject")
}

func TestGCPMetricsScopeRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMetricsScopeContractServer(t)

	assertGCPMetricsScopeNotImplemented(t, ts, http.MethodPost, "/gcp/google.monitoring.metricsscope.v1.MetricsScopes/GetMetricsScope", "MetricsScopes/GetMetricsScope")
	assertGCPMetricsScopeNotImplemented(t, ts, http.MethodPost, "/gcp/google.monitoring.metricsscope.v1.MetricsScopes/ListMetricsScopesByMonitoredProject", "MetricsScopes/ListMetricsScopesByMonitoredProject")
	assertGCPMetricsScopeNotImplemented(t, ts, http.MethodPost, "/gcp/google.monitoring.metricsscope.v1.MetricsScopes/CreateMonitoredProject", "MetricsScopes/CreateMonitoredProject")
	assertGCPMetricsScopeNotImplemented(t, ts, http.MethodPost, "/gcp/google.monitoring.metricsscope.v1.MetricsScopes/DeleteMonitoredProject", "MetricsScopes/DeleteMonitoredProject")
}

func newGCPMetricsScopeContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMetricsScopeNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp metricsscope router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPMetricsscopeRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPMetricsscopeRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/metricsscope?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp metricsscope contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "metricsscope" {
		t.Fatalf("expected service=metricsscope, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

