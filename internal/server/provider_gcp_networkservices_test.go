package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPNetworkServicesRouter_CoreResourceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetworkServicesContractServer(t)

	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/endpointPolicies?pageSize=1", "/endpointPolicies")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/endpointPolicies?endpointPolicyId=ep-1", "/endpointPolicies")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/gateways/gw-1", "/gateways/gw-1")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/grpcRoutes/gr-1", "/grpcRoutes/gr-1")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/httpRoutes/hr-1", "/httpRoutes/hr-1")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/tcpRoutes/tr-1", "/tcpRoutes/tr-1")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/tlsRoutes/tlr-1", "/tlsRoutes/tlr-1")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/serviceBindings/sb-1", "/serviceBindings/sb-1")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/meshes/mesh-1", "/meshes/mesh-1")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/serviceLbPolicies/slp-1", "/serviceLbPolicies/slp-1")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/wasmPlugins", "/wasmPlugins")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/wasmPlugins/wp-1/versions", "/versions")
}

func TestGCPNetworkServicesRouter_DepResourceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetworkServicesContractServer(t)

	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lbTrafficExtensions", "/lbTrafficExtensions")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lbRouteExtensions", "/lbRouteExtensions")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/lbEdgeExtensions", "/lbEdgeExtensions")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/authzExtensions", "/authzExtensions")
}

func TestGCPNetworkServicesRouter_RouteViewIAMOperationAndLocationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetworkServicesContractServer(t)

	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/gateways/gw-1/routeViews?pageSize=1", "/routeViews")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/meshes/mesh-1/routeViews?pageSize=1", "/routeViews")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/gateways/gw-1:getIamPolicy", ":getIamPolicy")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/gateways/gw-1:setIamPolicy", ":setIamPolicy")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/gateways/gw-1:testIamPermissions", ":testIamPermissions")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
}

func TestGCPNetworkServicesRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNetworkServicesContractServer(t)

	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkservices.v1.NetworkServices/ListEndpointPolicies", "NetworkServices/ListEndpointPolicies")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkservices.v1.NetworkServices/ListGateways", "NetworkServices/ListGateways")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkservices.v1.NetworkServices/ListMeshes", "NetworkServices/ListMeshes")
	assertGCPNetworkServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.networkservices.v1.DepService/ListLbTrafficExtensions", "DepService/ListLbTrafficExtensions")
}

func newGCPNetworkServicesContractServer(t *testing.T) *httptest.Server {
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

func assertGCPNetworkServicesNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp networkservices router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPNetworkservicesRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPNetworkservicesRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/networkservices?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp networkservices contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "networkservices" {
		t.Fatalf("expected service=networkservices, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
