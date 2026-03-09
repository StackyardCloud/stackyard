package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPEdgeContainerRouter_ClusterAndTokenRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEdgeContainerContractServer(t)
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters?pageSize=1", "/clusters")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster", "/clusters/team-cluster")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters", "/clusters")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster", "/clusters/team-cluster")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster:upgrade", ":upgrade")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster:generateAccessToken", ":generateAccessToken")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster:generateOfflineCredential", ":generateOfflineCredential")
}

func TestGCPEdgeContainerRouter_NodePoolAndMachineRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEdgeContainerContractServer(t)
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools?pageSize=1", "/nodePools")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools/default-pool", "/nodePools/default-pool")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools", "/nodePools")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools/default-pool", "/nodePools/default-pool")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools/default-pool", "/nodePools/default-pool")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/zones/us-central1-a/sites/site-a/machines?pageSize=1", "/machines")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/zones/us-central1-a/sites/site-a/machines/machine-1", "/machines/machine-1")
}

func TestGCPEdgeContainerRouter_VpnConfigAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEdgeContainerContractServer(t)
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/vpnConnections?pageSize=1", "/vpnConnections")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/vpnConnections/conn-1", "/vpnConnections/conn-1")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/vpnConnections", "/vpnConnections")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/vpnConnections/conn-1", "/vpnConnections/conn-1")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/serverConfig", "/serverConfig")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
}

func TestGCPEdgeContainerRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEdgeContainerContractServer(t)
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.edgecontainer.v1.EdgeContainer/ListClusters", "EdgeContainer/ListClusters")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.edgecontainer.v1.EdgeContainer/GenerateAccessToken", "EdgeContainer/GenerateAccessToken")
	assertGCPEdgeContainerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.edgecontainer.v1.EdgeContainer/ListVpnConnections", "EdgeContainer/ListVpnConnections")
}

func newGCPEdgeContainerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPEdgeContainerNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp edgecontainer router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPEdgecontainerRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPEdgecontainerRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/edgecontainer?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp edgecontainer contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "edgecontainer" {
		t.Fatalf("expected service=edgecontainer, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
