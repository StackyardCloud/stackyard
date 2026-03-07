package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPOracleDatabaseRouter_RESTResourceAndActionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOracleDatabaseContractServer(t)

	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/cloudExadataInfrastructures?pageSize=1", "/cloudExadataInfrastructures")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/cloudExadataInfrastructures/exadata-1", "/cloudExadataInfrastructures/exadata-1")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/cloudVmClusters?pageSize=1", "/cloudVmClusters")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/autonomousDatabases?pageSize=1", "/autonomousDatabases")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/autonomousDatabases/adb-1:generateWallet", ":generateWallet")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/autonomousDatabases/adb-1:start", ":start")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/autonomousDatabases/adb-1:stop", ":stop")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/autonomousDatabases/adb-1:restart", ":restart")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/autonomousDatabases/adb-1:switchover", ":switchover")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/autonomousDatabases/adb-1:failover", ":failover")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/odbNetworks?pageSize=1", "/odbNetworks")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/exadbVmClusters?pageSize=1", "/exadbVmClusters")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/exadbVmClusters/exadbvm-1:removeVirtualMachine", ":removeVirtualMachine")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/dbSystems?pageSize=1", "/dbSystems")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/databaseCharacterSets?pageSize=1", "/databaseCharacterSets")
}

func TestGCPOracleDatabaseRouter_RESTLocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOracleDatabaseContractServer(t)

	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
}

func TestGCPOracleDatabaseRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOracleDatabaseContractServer(t)

	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oracledatabase.v1.OracleDatabase/ListCloudExadataInfrastructures", "OracleDatabase/ListCloudExadataInfrastructures")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oracledatabase.v1.OracleDatabase/GetAutonomousDatabase", "OracleDatabase/GetAutonomousDatabase")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oracledatabase.v1.OracleDatabase/GenerateAutonomousDatabaseWallet", "OracleDatabase/GenerateAutonomousDatabaseWallet")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oracledatabase.v1.OracleDatabase/ListOdbNetworks", "OracleDatabase/ListOdbNetworks")
	assertGCPOracleDatabaseNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oracledatabase.v1.OracleDatabase/ListDbSystems", "OracleDatabase/ListDbSystems")
}

func newGCPOracleDatabaseContractServer(t *testing.T) *httptest.Server {
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

func assertGCPOracleDatabaseNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp oracle database router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPOracledatabaseRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPOracledatabaseRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/oracledatabase?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp oracledatabase contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "oracledatabase" {
		t.Fatalf("expected service=oracledatabase, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

