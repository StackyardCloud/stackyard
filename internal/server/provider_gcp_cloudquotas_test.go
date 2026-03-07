package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCloudQuotasRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudQuotasContractServer(t)

	baseService := "/gcp/v1/projects/stackyard/locations/global/services/compute.googleapis.com/quotaInfos"
	basePreferences := "/gcp/v1/projects/stackyard/locations/global/quotaPreferences"

	assertGCPCloudQuotasSuccess(t, ts, http.MethodGet, baseService+"?pageSize=1", nil, "quotaInfos")
	assertGCPCloudQuotasSuccess(t, ts, http.MethodGet, baseService+"/CpusPerProjectPerRegion", nil, "CpusPerProjectPerRegion")
	assertGCPCloudQuotasSuccess(t, ts, http.MethodGet, basePreferences+"?pageSize=1", nil, "quotaPreferences")
	assertGCPCloudQuotasSuccess(t, ts, http.MethodGet, basePreferences+"/team-config", nil, "team-config")
	assertGCPCloudQuotasSuccess(t, ts, http.MethodPost, basePreferences+"?quotaPreferenceId=team-config", []byte(`{"quotaPreference":{"service":"compute.googleapis.com","quotaId":"CpusPerProjectPerRegion","quotaConfig":{"preferredValue":16}}}`), "preferredValue")
	assertGCPCloudQuotasSuccess(t, ts, http.MethodPatch, basePreferences+"/team-config", []byte(`{"quotaPreference":{"name":"projects/stackyard/locations/global/quotaPreferences/team-config","service":"compute.googleapis.com","quotaId":"CpusPerProjectPerRegion","quotaConfig":{"preferredValue":32}}}`), "32")
}

func TestGCPCloudQuotasRouter_ListQuotaInfosInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudQuotasContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/services/compute.googleapis.com/quotaInfos?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudquotas router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudQuotasRouter_CreateQuotaPreferenceRequiresServiceAndQuotaID(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudQuotasContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/quotaPreferences?quotaPreferenceId=team-config", []byte(`{"quotaPreference":{"quotaConfig":{"preferredValue":16}}}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudquotas router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPCloudQuotasContractServer(t *testing.T) *httptest.Server {
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

func assertGCPCloudQuotasSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp cloudquotas router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPCloudQuotasRouter_UpdateQuotaPreferenceNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudQuotasContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/global/quotaPreferences/team-config", []byte(`{"quotaPreference":{"name":"projects/stackyard/locations/global/quotaPreferences/other","service":"compute.googleapis.com","quotaId":"CpusPerProjectPerRegion","quotaConfig":{"preferredValue":32}}}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudquotas router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}
