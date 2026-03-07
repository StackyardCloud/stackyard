package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMonitoringDashboardRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMonitoringDashboardContractServer(t)
	base := "/gcp/v1/projects/stackyard/dashboards"

	assertGCPMonitoringDashboardSuccess(t, ts, http.MethodGet, base+"?pageSize=1", nil, "dashboards")
	assertGCPMonitoringDashboardSuccess(t, ts, http.MethodPost, base, []byte(`{"dashboard":{"displayName":"Stackyard Dashboard"}}`), "Stackyard Dashboard")
	assertGCPMonitoringDashboardSuccess(t, ts, http.MethodGet, base+"/dashboard-1", nil, "dashboard-1")
	assertGCPMonitoringDashboardSuccess(t, ts, http.MethodPatch, base+"/dashboard-1", []byte(`{"dashboard":{"displayName":"Updated Dashboard"}}`), "dashboard-1")
	assertGCPMonitoringDashboardSuccess(t, ts, http.MethodDelete, base+"/dashboard-1", nil, "{}")
	assertGCPMonitoringDashboardSuccess(t, ts, http.MethodGet, "/gcp/v1/dashboards/managed-dashboard-1", nil, "managed-dashboard-1")
}

func TestGCPMonitoringDashboardRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPMonitoringDashboardContractServer(t)
	assertGCPMonitoringDashboardNotImplemented(t, ts, http.MethodPost, "/gcp/google.monitoring.dashboard.v1.DashboardsService/ListDashboards", "DashboardsService/ListDashboards")
}

func TestGCPMonitoringDashboardRouter_ListDashboardsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPMonitoringDashboardContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/dashboards?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp monitoring dashboard router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func TestGCPMonitoringDashboardRouter_CreateRequiresDashboard(t *testing.T) {
	t.Parallel()

	ts := newGCPMonitoringDashboardContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/dashboards", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp monitoring dashboard router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPMonitoringDashboardContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMonitoringDashboardNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp monitoring dashboard router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPMonitoringDashboardSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp monitoring dashboard router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
