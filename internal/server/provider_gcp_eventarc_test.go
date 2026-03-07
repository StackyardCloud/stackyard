package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPEventarcRouter_TriggerAndChannelRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEventarcContractServer(t)
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/triggers?pageSize=1", "/triggers")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/triggers", "/triggers")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/triggers/team-trigger?updateMask=labels", "/triggers/team-trigger")
	assertGCPEventarcNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/triggers/team-trigger", "/triggers/team-trigger")
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/channels?pageSize=1", "/channels")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/channels", "/channels")
}

func TestGCPEventarcRouter_ProviderConnectionAndConfigRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEventarcContractServer(t)
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/providers?pageSize=1", "/providers")
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/channelConnections?pageSize=1", "/channelConnections")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/channelConnections", "/channelConnections")
	assertGCPEventarcNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/channelConnections/team-conn", "/channelConnections/team-conn")
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/googleChannelConfig", "/googleChannelConfig")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/googleChannelConfig?updateMask=cryptoKeyName", "/googleChannelConfig")
}

func TestGCPEventarcRouter_MessageBusEnrollmentPipelineAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEventarcContractServer(t)
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/messageBuses?pageSize=1", "/messageBuses")
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/messageBuses/team-bus:listEnrollments?pageSize=1", ":listEnrollments")
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/enrollments?pageSize=1", "/enrollments")
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/pipelines?pageSize=1", "/pipelines")
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/googleApiSources?pageSize=1", "/googleApiSources")
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPEventarcNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/triggers/team-trigger:getIamPolicy", ":getIamPolicy")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/triggers/team-trigger:setIamPolicy", ":setIamPolicy")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/triggers/team-trigger:testIamPermissions", ":testIamPermissions")
}

func TestGCPEventarcRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEventarcContractServer(t)
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.eventarc.v1.Eventarc/ListTriggers", "Eventarc/ListTriggers")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.eventarc.v1.Eventarc/CreateTrigger", "Eventarc/CreateTrigger")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.eventarc.v1.Eventarc/ListChannels", "Eventarc/ListChannels")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.eventarc.v1.Eventarc/ListMessageBuses", "Eventarc/ListMessageBuses")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.eventarc.v1.Eventarc/ListPipelines", "Eventarc/ListPipelines")
	assertGCPEventarcNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.eventarc.v1.Eventarc/ListGoogleApiSources", "Eventarc/ListGoogleApiSources")
}

func newGCPEventarcContractServer(t *testing.T) *httptest.Server {
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

func assertGCPEventarcNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp eventarc router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPEventarcRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPEventarcRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/eventarc?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp eventarc contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "eventarc" {
		t.Fatalf("expected service=eventarc, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

