package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCloudProfilerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudProfilerContractServer(t)

	assertGCPCloudProfilerSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/profiles?pageSize=1", nil, "profiles")
	assertGCPCloudProfilerSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/profiles/stackyard-profile", nil, "stackyard-profile")
	assertGCPCloudProfilerSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/profiles", []byte(`{"deployment":{"projectId":"stackyard","target":"stackyard-service"},"profileType":["CPU"]}`), "stackyard-profile")
	assertGCPCloudProfilerSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/profiles:createOffline", []byte(`{"profile":{"name":"projects/stackyard/profiles/offline-profile","profileType":"CPU","profileBytes":"c3RhY2t5YXJk"}}`), "offline-profile")
	assertGCPCloudProfilerSuccess(t, ts, http.MethodPatch, "/gcp/v2/projects/stackyard/profiles/stackyard-profile", []byte(`{"profile":{"name":"projects/stackyard/profiles/stackyard-profile","labels":{"updated-by":"test"}}}`), "updated-by")
}

func TestGCPCloudProfilerRouter_ListProfilesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudProfilerContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/profiles?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudprofiler router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudProfilerRouter_CreateProfileRequiresDeployment(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudProfilerContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/profiles", []byte(`{"profileType":["CPU"]}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudprofiler router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPCloudProfilerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPCloudProfilerSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp cloudprofiler router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
