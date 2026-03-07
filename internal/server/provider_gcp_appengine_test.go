package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPAppEngineRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPAppEngineContractServer(t)

	assertGCPAppEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/apps/stackyard", nil, `"name":"apps/stackyard"`)
	assertGCPAppEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/apps/stackyard/services?pageSize=1", nil, "services")
	assertGCPAppEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/apps/stackyard/services/default", nil, "services/default")
	assertGCPAppEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/apps/stackyard/services/default/versions?pageSize=1&view=BASIC", nil, "versions")
	assertGCPAppEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/apps/stackyard/services/default/versions/v1?view=BASIC", nil, "versions/v1")
	assertGCPAppEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/apps/stackyard/services/default/versions/v1/instances?pageSize=1", nil, "instances")
	assertGCPAppEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/apps/stackyard/services/default/versions/v1/instances/i-1", nil, "instances/i-1")
}

func TestGCPAppEngineRouter_ListServicesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPAppEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/apps/stackyard/services?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp appengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAppEngineRouter_GetVersionInvalidView(t *testing.T) {
	t.Parallel()

	ts := newGCPAppEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/apps/stackyard/services/default/versions/v1?view=INVALID", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp appengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAppEngineRouter_ListInstancesPageTokenOutOfRange(t *testing.T) {
	t.Parallel()

	ts := newGCPAppEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/apps/stackyard/services/default/versions/v1/instances?pageToken=99", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp appengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPAppEngineContractServer(t *testing.T) *httptest.Server {
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

func assertGCPAppEngineSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp appengine router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
