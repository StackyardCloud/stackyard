package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCloudDMSRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudDMSContractServer(t)
	base := "/gcp/v1/projects/stackyard/locations/us-central1"
	workspace := base + "/conversionWorkspaces/team-workspace"

	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, base+"/migrationJobs?pageSize=1", nil, "migrationJobs")
	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, base+"/migrationJobs/team-job", nil, "team-job")
	assertGCPCloudDMSSuccess(t, ts, http.MethodPost, base+"/migrationJobs?migrationJobId=team-job", []byte(`{"migrationJob":{"displayName":"Stackyard Migration Job"}}`), "operations")
	assertGCPCloudDMSSuccess(t, ts, http.MethodPost, base+"/migrationJobs/team-job:start", []byte(`{}`), "operations")

	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, base+"/connectionProfiles?pageSize=1", nil, "connectionProfiles")
	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, base+"/connectionProfiles/team-profile", nil, "team-profile")
	assertGCPCloudDMSSuccess(t, ts, http.MethodPost, base+"/connectionProfiles?connectionProfileId=team-profile", []byte(`{"connectionProfile":{"displayName":"Stackyard Connection Profile"}}`), "operations")

	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, base+"/privateConnections?pageSize=1", nil, "privateConnections")
	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, base+"/privateConnections/team-private-connection", nil, "team-private-connection")
	assertGCPCloudDMSSuccess(t, ts, http.MethodPost, base+"/privateConnections?privateConnectionId=team-private-connection", []byte(`{"privateConnection":{"displayName":"Stackyard Private Connection"}}`), "operations")

	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, base+"/conversionWorkspaces?pageSize=1", nil, "conversionWorkspaces")
	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, base+"/conversionWorkspaces/team-workspace", nil, "team-workspace")
	assertGCPCloudDMSSuccess(t, ts, http.MethodPost, base+"/conversionWorkspaces?conversionWorkspaceId=team-workspace", []byte(`{"conversionWorkspace":{"displayName":"Stackyard Conversion Workspace"}}`), "operations")

	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, workspace+"/mappingRules?pageSize=1", nil, "mappingRules")
	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, workspace+"/mappingRules/team-rule", nil, "team-rule")
	assertGCPCloudDMSSuccess(t, ts, http.MethodPost, workspace+"/mappingRules?mappingRuleId=team-rule", []byte(`{"mappingRule":{"ruleOrder":1}}`), "operations")

	assertGCPCloudDMSSuccess(t, ts, http.MethodPost, workspace+":describeDatabaseEntities?pageSize=1", []byte(`{}`), "databaseEntities")
	assertGCPCloudDMSSuccess(t, ts, http.MethodPost, workspace+":searchBackgroundJobs", []byte(`{"maxSize":1}`), "jobs")
	assertGCPCloudDMSSuccess(t, ts, http.MethodPost, workspace+":describeConversionWorkspaceRevisions", []byte(`{}`), "revisions")

	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, base+":fetchStaticIps?pageSize=1", nil, "staticIps")
	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, base+"/operations?pageSize=1", nil, "operations")
	assertGCPCloudDMSSuccess(t, ts, http.MethodGet, base+"/operations/team-operation", nil, "team-operation")
}

func TestGCPCloudDMSRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudDMSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.clouddms.v1.DataMigrationService/ListMigrationJobs", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp clouddms router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, "DataMigrationService/ListMigrationJobs") {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudDMSRouter_ListMigrationJobsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudDMSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/migrationJobs?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp clouddms router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudDMSRouter_CreateMigrationJobRequiresBody(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudDMSContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/migrationJobs?migrationJobId=team-job", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp clouddms router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPCloudDMSContractServer(t *testing.T) *httptest.Server {
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

func assertGCPCloudDMSSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "clouddms",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp clouddms router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
