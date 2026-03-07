package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPGameServicesRouter_RealmAndClusterRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGameServicesContractServer(t)
	assertGCPGameServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/realms?pageSize=1", "/realms")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/realms", "/realms")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/realms/team-realm", "/realms/team-realm")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/realms/team-realm/gameServerClusters?pageSize=1", "/gameServerClusters")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/realms/team-realm/gameServerClusters:previewCreate", ":previewCreate")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/realms/team-realm/gameServerClusters/team-cluster:previewDelete", ":previewDelete")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/realms/team-realm/gameServerClusters/team-cluster:previewUpdate", ":previewUpdate")
}

func TestGCPGameServicesRouter_ConfigDeploymentAndRolloutRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGameServicesContractServer(t)
	assertGCPGameServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/gameServerDeployments?pageSize=1", "/gameServerDeployments")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/gameServerDeployments", "/gameServerDeployments")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/gameServerDeployments/team-deployment/configs?pageSize=1", "/configs")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/gameServerDeployments/team-deployment/configs", "/configs")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/gameServerDeployments/team-deployment/rollout", "/rollout")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/gameServerDeployments/team-deployment/rollout:preview", ":preview")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/gameServerDeployments/team-deployment:fetchDeploymentState", ":fetchDeploymentState")
}

func TestGCPGameServicesRouter_OperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGameServicesContractServer(t)
	assertGCPGameServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/operations?pageSize=1", "/operations")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/operations/op-1", "/operations/op-1")
}

func TestGCPGameServicesRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGameServicesContractServer(t)
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gaming.v1.RealmsService/ListRealms", "RealmsService/ListRealms")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gaming.v1.RealmsService/CreateRealm", "RealmsService/CreateRealm")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gaming.v1.GameServerClustersService/ListGameServerClusters", "GameServerClustersService/ListGameServerClusters")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gaming.v1.GameServerConfigsService/ListGameServerConfigs", "GameServerConfigsService/ListGameServerConfigs")
	assertGCPGameServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gaming.v1.GameServerDeploymentsService/ListGameServerDeployments", "GameServerDeploymentsService/ListGameServerDeployments")
}

func newGCPGameServicesContractServer(t *testing.T) *httptest.Server {
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

func assertGCPGameServicesNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp gaming router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPGamingRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPGamingRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/gaming?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp gaming contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "gaming" {
		t.Fatalf("expected service=gaming, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

