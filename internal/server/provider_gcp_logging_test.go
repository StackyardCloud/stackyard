package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPLoggingRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLoggingContractServer(t)
	project := "/gcp/v2/projects/stackyard"

	assertGCPLoggingSuccess(t, ts, http.MethodPost, "/gcp/v2/entries:write", []byte(`{"entries":[{"textPayload":"hello"}]}`), "logEntryErrors")
	assertGCPLoggingSuccess(t, ts, http.MethodPost, "/gcp/v2/entries:list", []byte(`{"resourceNames":["projects/stackyard"],"pageSize":1}`), "entries")

	assertGCPLoggingSuccess(t, ts, http.MethodGet, project+"/logs?pageSize=1", nil, "logNames")
	assertGCPLoggingSuccess(t, ts, http.MethodDelete, project+"/logs/stackyard%2Fapp", nil, "{}")
	assertGCPLoggingSuccess(t, ts, http.MethodGet, project+"/monitoredResourceDescriptors?pageSize=1", nil, "resourceDescriptors")

	assertGCPLoggingSuccess(t, ts, http.MethodGet, project+"/sinks?pageSize=1", nil, "sinks")
	assertGCPLoggingSuccess(t, ts, http.MethodGet, project+"/sinks/export-a", nil, "export-a")
	assertGCPLoggingSuccess(t, ts, http.MethodPost, project+"/sinks", []byte(`{"sink":{"name":"export-a","destination":"storage.googleapis.com/stackyard-logs"}}`), "export-a")
	assertGCPLoggingSuccess(t, ts, http.MethodDelete, project+"/sinks/export-a", nil, "{}")

	assertGCPLoggingSuccess(t, ts, http.MethodGet, project+"/exclusions?pageSize=1", nil, "exclusions")
	assertGCPLoggingSuccess(t, ts, http.MethodGet, project+"/exclusions/exclude-debug", nil, "exclude-debug")
	assertGCPLoggingSuccess(t, ts, http.MethodPost, project+"/exclusions", []byte(`{"exclusion":{"name":"exclude-debug","filter":"severity=DEBUG"}}`), "exclude-debug")
	assertGCPLoggingSuccess(t, ts, http.MethodDelete, project+"/exclusions/exclude-debug", nil, "{}")

	assertGCPLoggingSuccess(t, ts, http.MethodGet, project+"/metrics?pageSize=1", nil, "metrics")
	assertGCPLoggingSuccess(t, ts, http.MethodGet, project+"/metrics/error_count", nil, "error_count")
	assertGCPLoggingSuccess(t, ts, http.MethodPost, project+"/metrics", []byte(`{"metric":{"name":"error_count","filter":"severity>=ERROR"}}`), "error_count")
	assertGCPLoggingSuccess(t, ts, http.MethodDelete, project+"/metrics/error_count", nil, "{}")

	assertGCPLoggingSuccess(t, ts, http.MethodGet, project+"/locations/us-central1/buckets?pageSize=1", nil, "buckets")
}

func TestGCPLoggingRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPLoggingContractServer(t)
	assertGCPLoggingNotImplemented(t, ts, http.MethodPost, "/gcp/google.logging.v2.LoggingServiceV2/ListLogEntries", "LoggingServiceV2/ListLogEntries")
}

func TestGCPLoggingRouter_WriteEntriesRequiresEntries(t *testing.T) {
	t.Parallel()

	ts := newGCPLoggingContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/entries:write", []byte(`{"entries":[]}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp logging router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func TestGCPLoggingRouter_ListLogsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPLoggingContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/logs?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp logging router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPLoggingContractServer(t *testing.T) *httptest.Server {
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

func assertGCPLoggingNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp logging router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPLoggingSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp logging router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
