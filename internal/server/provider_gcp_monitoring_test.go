package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMonitoringRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMonitoringContractServer(t)
	project := "/gcp/v3/projects/stackyard"

	assertGCPMonitoringSuccess(t, ts, http.MethodGet, project+"/alertPolicies?pageSize=1", nil, "alertPolicies")
	assertGCPMonitoringSuccess(t, ts, http.MethodPost, project+"/alertPolicies", []byte(`{"alertPolicy":{"displayName":"stackyard alert policy"}}`), "alertPolicies")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, project+"/groups?pageSize=1", nil, "group")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, project+"/groups/group-1/members?pageSize=1", nil, "members")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, project+"/metricDescriptors?pageSize=1", nil, "metricDescriptors")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, project+"/monitoredResourceDescriptors?pageSize=1", nil, "resourceDescriptors")
	assertGCPMonitoringSuccess(t, ts, http.MethodPost, project+"/timeSeries:query", []byte(`{"query":"fetch cpu"}`), "timeSeriesData")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, project+"/notificationChannelDescriptors?pageSize=1", nil, "channelDescriptors")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, project+"/notificationChannels?pageSize=1", nil, "notificationChannels")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, project+"/services?pageSize=1", nil, "services")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, project+"/services/service-a/serviceLevelObjectives?pageSize=1", nil, "serviceLevelObjectives")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, project+"/snoozes?pageSize=1", nil, "snoozes")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, project+"/uptimeCheckConfigs?pageSize=1", nil, "uptimeCheckConfigs")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, "/gcp/v3/uptimeCheckIps?pageSize=1", nil, "uptimeCheckIps")

	workspace := "/gcp/v3/workspaces/stackyard-host"
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, workspace+"/services?pageSize=1", nil, "services")
	assertGCPMonitoringSuccess(t, ts, http.MethodGet, workspace+"/services/-/serviceLevelObjectives?pageSize=1", nil, "serviceLevelObjectives")
}

func TestGCPMonitoringRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPMonitoringContractServer(t)
	assertGCPMonitoringNotImplemented(t, ts, http.MethodPost, "/gcp/google.monitoring.v3.AlertPolicyService/ListAlertPolicies", "AlertPolicyService/ListAlertPolicies")
}

func TestGCPMonitoringRouter_QueryTimeSeriesRequiresQuery(t *testing.T) {
	t.Parallel()

	ts := newGCPMonitoringContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/timeSeries:query", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp monitoring router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func TestGCPMonitoringRouter_ListAlertPoliciesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPMonitoringContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/alertPolicies?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp monitoring router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPMonitoringContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMonitoringNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp monitoring router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPMonitoringSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp monitoring router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
