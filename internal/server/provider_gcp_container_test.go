package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPContainerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPContainerContractServer(t)

	assertGCPContainerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters?pageSize=1", nil, "team-cluster")
	assertGCPContainerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster", nil, "RUNNING")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters", []byte(`{"cluster":{"name":"team-cluster"}}`), "CREATE_CLUSTER")
	assertGCPContainerSuccess(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster", nil, "DELETE_CLUSTER")
	assertGCPContainerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools?pageSize=1", nil, "default-pool")
	assertGCPContainerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools/default-pool", nil, "RUNNING")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools", []byte(`{"nodePool":{"name":"default-pool"}}`), "CREATE_NODE_POOL")
	assertGCPContainerSuccess(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools/default-pool", nil, "DELETE_NODE_POOL")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster:setLogging", []byte(`{"loggingService":"logging.googleapis.com/kubernetes"}`), "SET_LOGGING")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster:setMonitoring", []byte(`{"monitoringService":"monitoring.googleapis.com/kubernetes"}`), "SET_MONITORING")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster:setAddons", []byte(`{"addonsConfig":{}}`), "SET_ADDONS")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools/default-pool:setManagement", []byte(`{"management":{"autoRepair":true}}`), "SET_NODE_POOL_MANAGEMENT")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools/default-pool:setSize", []byte(`{"nodeCount":1}`), "SET_NODE_POOL_SIZE")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools/default-pool:completeUpgrade", []byte(`{}`), "{}")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools/default-pool:rollback", []byte(`{}`), "ROLLBACK_NODE_POOL")
	assertGCPContainerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", nil, "op-1")
	assertGCPContainerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", nil, "op-1")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", []byte(`{}`), "{}")
	assertGCPContainerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/serverConfig", nil, "defaultClusterVersion")
	assertGCPContainerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/aggregated/usableSubnetworks?pageSize=1", nil, "subnetworks")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster:checkAutopilotCompatibility", []byte(`{}`), "{}")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster:fetchClusterUpgradeInfo", []byte(`{"version":"v1"}`), "{}")
	assertGCPContainerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools/default-pool:fetchNodePoolUpgradeInfo", []byte(`{"version":"v1"}`), "{}")
}

func TestGCPContainerRouter_ListClustersInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPContainerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "container",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp container router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPContainerRouter_CreateClusterRequiresName(t *testing.T) {
	t.Parallel()

	ts := newGCPContainerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters", []byte(`{"cluster":{}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "container",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp container router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPContainerRouter_SetNodePoolSizeRequiresNumericNodeCount(t *testing.T) {
	t.Parallel()

	ts := newGCPContainerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/team-cluster/nodePools/default-pool:setSize", []byte(`{"nodeCount":"one"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "container",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp container router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPContainerRouter_ListUsableSubnetworksPageTokenOutOfRange(t *testing.T) {
	t.Parallel()

	ts := newGCPContainerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/aggregated/usableSubnetworks?pageToken=99", nil, map[string]string{
		"X-Stackyard-GCP-Service": "container",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp container router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPContainerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPContainerSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	headers["X-Stackyard-GCP-Service"] = "container"
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp container router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
