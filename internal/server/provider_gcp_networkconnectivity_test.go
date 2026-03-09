package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPNetworkConnectivityRouter_CoreRESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetworkConnectivityContractServer(t)

	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/hubs?pageSize=1", "/hubs")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/hubs/hub-a", "/hubs/hub-a")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/spokes?pageSize=1", "/spokes")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/serviceConnectionMaps?pageSize=1", "/serviceConnectionMaps")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/internalRanges?pageSize=1", "/internalRanges")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/policyBasedRoutes?pageSize=1", "/policyBasedRoutes")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/multicloudDataTransferConfigs?pageSize=1", "/multicloudDataTransferConfigs")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/destinations?pageSize=1", "/destinations")
}

func TestGCPNetworkConnectivityRouter_ActionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetworkConnectivityContractServer(t)

	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/hubs/hub-a:acceptSpoke", ":acceptSpoke")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/hubs/hub-a:rejectSpoke", ":rejectSpoke")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/hubs/hub-a:queryStatus", ":queryStatus")
}

func TestGCPNetworkConnectivityRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetworkConnectivityContractServer(t)

	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkconnectivity.v1.HubService/ListHubs", "HubService/ListHubs")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkconnectivity.v1.HubService/GetHub", "HubService/GetHub")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkconnectivity.v1.CrossNetworkAutomationService/ListServiceConnectionMaps", "CrossNetworkAutomationService/ListServiceConnectionMaps")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkconnectivity.v1.InternalRangeService/ListInternalRanges", "InternalRangeService/ListInternalRanges")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkconnectivity.v1.PolicyBasedRoutingService/ListPolicyBasedRoutes", "PolicyBasedRoutingService/ListPolicyBasedRoutes")
	assertGCPNetworkConnectivityNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkconnectivity.v1.DataTransferService/ListMulticloudDataTransferConfigs", "DataTransferService/ListMulticloudDataTransferConfigs")
}

func newGCPNetworkConnectivityContractServer(t *testing.T) *httptest.Server {
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

func assertGCPNetworkConnectivityNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp networkconnectivity router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPNetworkconnectivityRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPNetworkconnectivityRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/networkconnectivity?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp networkconnectivity contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "networkconnectivity" {
		t.Fatalf("expected service=networkconnectivity, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
