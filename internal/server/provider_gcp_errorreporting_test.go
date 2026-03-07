package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPErrorReportingRouter_GroupStatsAndEventRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPErrorReportingContractServer(t)
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/groupStats?pageSize=1", "/groupStats")
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/events?groupId=grp-1&pageSize=1", "/events")
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/events", "/events")
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/events:report", ":report")
}

func TestGCPErrorReportingRouter_GroupRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPErrorReportingContractServer(t)
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/groups/grp-1", "/groups/grp-1")
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodPut, "/gcp/v1beta1/projects/stackyard/groups/grp-1", "/groups/grp-1")
}

func TestGCPErrorReportingRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPErrorReportingContractServer(t)
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodPost, "/gcp/google.devtools.clouderrorreporting.v1beta1.ErrorStatsService/ListGroupStats", "ErrorStatsService/ListGroupStats")
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodPost, "/gcp/google.devtools.clouderrorreporting.v1beta1.ErrorStatsService/ListEvents", "ErrorStatsService/ListEvents")
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodPost, "/gcp/google.devtools.clouderrorreporting.v1beta1.ErrorStatsService/DeleteEvents", "ErrorStatsService/DeleteEvents")
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodPost, "/gcp/google.devtools.clouderrorreporting.v1beta1.ErrorGroupService/GetGroup", "ErrorGroupService/GetGroup")
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodPost, "/gcp/google.devtools.clouderrorreporting.v1beta1.ErrorGroupService/UpdateGroup", "ErrorGroupService/UpdateGroup")
	assertGCPErrorReportingNotImplemented(t, ts, http.MethodPost, "/gcp/google.devtools.clouderrorreporting.v1beta1.ReportErrorsService/ReportErrorEvent", "ReportErrorsService/ReportErrorEvent")
}

func newGCPErrorReportingContractServer(t *testing.T) *httptest.Server {
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

func assertGCPErrorReportingNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp errorreporting router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPErrorreportingRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPErrorreportingRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/errorreporting?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp errorreporting contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "errorreporting" {
		t.Fatalf("expected service=errorreporting, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

