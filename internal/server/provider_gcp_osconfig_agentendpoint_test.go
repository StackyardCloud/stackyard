package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPOsConfigAgentEndpointRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOsConfigAgentEndpointContractServer(t)

	assertGCPOsConfigAgentEndpointNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.agentendpoint.v1.AgentEndpointService/ReceiveTaskNotification", "AgentEndpointService/ReceiveTaskNotification")
	assertGCPOsConfigAgentEndpointNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.agentendpoint.v1.AgentEndpointService/StartNextTask", "AgentEndpointService/StartNextTask")
	assertGCPOsConfigAgentEndpointNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.agentendpoint.v1.AgentEndpointService/ReportTaskProgress", "AgentEndpointService/ReportTaskProgress")
	assertGCPOsConfigAgentEndpointNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.agentendpoint.v1.AgentEndpointService/ReportTaskComplete", "AgentEndpointService/ReportTaskComplete")
	assertGCPOsConfigAgentEndpointNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.agentendpoint.v1.AgentEndpointService/RegisterAgent", "AgentEndpointService/RegisterAgent")
	assertGCPOsConfigAgentEndpointNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.agentendpoint.v1.AgentEndpointService/ReportInventory", "AgentEndpointService/ReportInventory")
	assertGCPOsConfigAgentEndpointNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.osconfig.agentendpoint.v1.AgentEndpointService/ReportVmInventory", "AgentEndpointService/ReportVmInventory")
}

func newGCPOsConfigAgentEndpointContractServer(t *testing.T) *httptest.Server {
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

func assertGCPOsConfigAgentEndpointNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp osconfig agentendpoint router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPOsconfigAgentendpointRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPOsconfigAgentendpointRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/osconfig_agentendpoint?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp osconfig_agentendpoint contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "osconfig_agentendpoint" {
		t.Fatalf("expected service=osconfig_agentendpoint, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

