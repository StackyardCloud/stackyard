package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPComputeRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPComputeContractServer(t)

	assertGCPComputeSuccess(t, ts, http.MethodGet, "/gcp/compute/v1/projects/stackyard/zones?pageSize=1", nil, "zones/us-central1-a")
	assertGCPComputeSuccess(t, ts, http.MethodGet, "/gcp/compute/v1/projects/stackyard/zones/us-central1-a", nil, "us-central1-a")
	assertGCPComputeSuccess(t, ts, http.MethodGet, "/gcp/compute/v1/projects/stackyard/global/networks?pageSize=1", nil, "team-network")
	assertGCPComputeSuccess(t, ts, http.MethodGet, "/gcp/compute/v1/projects/stackyard/global/networks/team-network", nil, "team-network")
	assertGCPComputeSuccess(t, ts, http.MethodPost, "/gcp/compute/v1/projects/stackyard/global/networks", []byte(`{"name":"team-network"}`), "insertNetwork")
	assertGCPComputeSuccess(t, ts, http.MethodGet, "/gcp/compute/v1/projects/stackyard/zones/us-central1-a/instances?pageSize=1", nil, "team-vm")
	assertGCPComputeSuccess(t, ts, http.MethodGet, "/gcp/compute/v1/projects/stackyard/zones/us-central1-a/instances/team-vm", nil, "RUNNING")
	assertGCPComputeSuccess(t, ts, http.MethodPost, "/gcp/compute/v1/projects/stackyard/zones/us-central1-a/instances", []byte(`{"name":"team-vm"}`), "insertInstance")
	assertGCPComputeSuccess(t, ts, http.MethodPost, "/gcp/compute/v1/projects/stackyard/zones/us-central1-a/instances/team-vm/start", nil, "\"operationType\":\"start\"")
	assertGCPComputeSuccess(t, ts, http.MethodPost, "/gcp/compute/v1/projects/stackyard/zones/us-central1-a/instances/team-vm/stop", nil, "\"operationType\":\"stop\"")
	assertGCPComputeSuccess(t, ts, http.MethodDelete, "/gcp/compute/v1/projects/stackyard/zones/us-central1-a/instances/team-vm", nil, "\"operationType\":\"delete\"")
	assertGCPComputeSuccess(t, ts, http.MethodGet, "/gcp/compute/v1/projects/stackyard/zones/us-central1-a/operations?pageSize=1", nil, "op-zone-1")
	assertGCPComputeSuccess(t, ts, http.MethodGet, "/gcp/compute/v1/projects/stackyard/zones/us-central1-a/operations/op-zone-1", nil, "op-zone-1")
	assertGCPComputeSuccess(t, ts, http.MethodPost, "/gcp/compute/v1/projects/stackyard/zones/us-central1-a/operations/op-zone-1/wait", nil, "\"status\":\"DONE\"")
	assertGCPComputeSuccess(t, ts, http.MethodGet, "/gcp/compute/v1/projects/stackyard/global/operations/op-global-1", nil, "op-global-1")
	assertGCPComputeSuccess(t, ts, http.MethodPost, "/gcp/compute/v1/projects/stackyard/global/operations/op-global-1/wait", nil, "\"status\":\"DONE\"")
}

func TestGCPComputeRouter_ListZonesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPComputeContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/compute/v1/projects/stackyard/zones?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp compute router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPComputeRouter_ListNetworksPageTokenOutOfRange(t *testing.T) {
	t.Parallel()

	ts := newGCPComputeContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/compute/v1/projects/stackyard/global/networks?pageToken=99", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp compute router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPComputeRouter_InsertNetworkRequiresName(t *testing.T) {
	t.Parallel()

	ts := newGCPComputeContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/compute/v1/projects/stackyard/global/networks", []byte(`{"routingConfig":{"routingMode":"REGIONAL"}}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp compute router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPComputeRouter_InsertInstanceInvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPComputeContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/compute/v1/projects/stackyard/zones/us-central1-a/instances", []byte(`{"name":`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp compute router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPComputeContractServer(t *testing.T) *httptest.Server {
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

func assertGCPComputeSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp compute router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
