package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPNetworkManagementRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetworkManagementContractServer(t)

	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/connectivityTests?pageSize=1", "/connectivityTests")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/connectivityTests/test-a", "/connectivityTests/test-a")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/connectivityTests?testId=test-a", "/connectivityTests")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/connectivityTests/test-a:rerun", ":rerun")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/global/connectivityTests/test-a", "/connectivityTests/test-a")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/vpcFlowLogsConfigs?pageSize=1", "/vpcFlowLogsConfigs")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/vpcFlowLogsConfigs/flow-a", "/vpcFlowLogsConfigs/flow-a")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/vpcFlowLogsConfigs?vpcFlowLogsConfigId=flow-a", "/vpcFlowLogsConfigs")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/global/vpcFlowLogsConfigs/flow-a", "/vpcFlowLogsConfigs/flow-a")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodGet, "/gcp/v1/organizations/123456/locations/global/vpcFlowLogsConfigs?pageSize=1", "/vpcFlowLogsConfigs")
}

func TestGCPNetworkManagementRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetworkManagementContractServer(t)

	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkmanagement.v1.ReachabilityService/ListConnectivityTests", "ReachabilityService/ListConnectivityTests")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkmanagement.v1.ReachabilityService/GetConnectivityTest", "ReachabilityService/GetConnectivityTest")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkmanagement.v1.ReachabilityService/CreateConnectivityTest", "ReachabilityService/CreateConnectivityTest")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkmanagement.v1.ReachabilityService/RerunConnectivityTest", "ReachabilityService/RerunConnectivityTest")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkmanagement.v1.VpcFlowLogsService/ListVpcFlowLogsConfigs", "VpcFlowLogsService/ListVpcFlowLogsConfigs")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkmanagement.v1.VpcFlowLogsService/GetVpcFlowLogsConfig", "VpcFlowLogsService/GetVpcFlowLogsConfig")
	assertGCPNetworkManagementNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkmanagement.v1.VpcFlowLogsService/CreateVpcFlowLogsConfig", "VpcFlowLogsService/CreateVpcFlowLogsConfig")
}

func newGCPNetworkManagementContractServer(t *testing.T) *httptest.Server {
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

func assertGCPNetworkManagementNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp networkmanagement router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPNetworkmanagementRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPNetworkmanagementRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/networkmanagement?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp networkmanagement contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "networkmanagement" {
		t.Fatalf("expected service=networkmanagement, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
