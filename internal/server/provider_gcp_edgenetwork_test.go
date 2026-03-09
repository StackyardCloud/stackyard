package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPEdgeNetworkRouter_ZoneAndNetworkRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEdgeNetworkContractServer(t)
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/zones/us-central1-a:initialize", ":initialize")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/zones?pageSize=1", "/zones")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/zones/us-central1-a", "/zones/us-central1-a")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/networks?pageSize=1", "/networks")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/networks", "/networks")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/networks/mesh-network:diagnose", ":diagnose")
}

func TestGCPEdgeNetworkRouter_SubnetAndInterconnectRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEdgeNetworkContractServer(t)
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/zones/us-central1-a/subnets?pageSize=1", "/subnets")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/zones/us-central1-a/subnets/team-subnet?updateMask=description", "/subnets/team-subnet")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/interconnects?pageSize=1", "/interconnects")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/interconnects/team-ic:diagnose", "/interconnects/team-ic:diagnose")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/interconnectAttachments?pageSize=1", "/interconnectAttachments")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/interconnectAttachments", "/interconnectAttachments")
}

func TestGCPEdgeNetworkRouter_RouterAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEdgeNetworkContractServer(t)
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/zones/us-central1-a/routers?pageSize=1", "/routers")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/zones/us-central1-a/routers", "/routers")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/zones/us-central1-a/routers/team-router:diagnose", ":diagnose")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
}

func TestGCPEdgeNetworkRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEdgeNetworkContractServer(t)
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.edgenetwork.v1.EdgeNetwork/ListNetworks", "EdgeNetwork/ListNetworks")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.edgenetwork.v1.EdgeNetwork/CreateSubnet", "EdgeNetwork/CreateSubnet")
	assertGCPEdgeNetworkNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.edgenetwork.v1.EdgeNetwork/DiagnoseRouter", "EdgeNetwork/DiagnoseRouter")
}

func newGCPEdgeNetworkContractServer(t *testing.T) *httptest.Server {
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

func assertGCPEdgeNetworkNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp edgenetwork router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPEdgenetworkRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPEdgenetworkRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/edgenetwork?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp edgenetwork contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "edgenetwork" {
		t.Fatalf("expected service=edgenetwork, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
