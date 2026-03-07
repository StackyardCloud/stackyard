package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPRedisClusterRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisClusterContractServer(t)

	base := "/gcp/v1/projects/stackyard/locations/us-central1"
	cluster := base + "/clusters/cluster-1"
	backupCollection := base + "/backupCollections/collection-1"
	backup := backupCollection + "/backups/backup-1"
	operation := base + "/operations/op-1"

	assertGCPRedisClusterSuccess(t, ts, http.MethodGet, base+"/clusters?pageSize=1", nil, "clusters")
	assertGCPRedisClusterSuccess(t, ts, http.MethodGet, cluster, nil, "clusters/cluster-1")
	assertGCPRedisClusterSuccess(t, ts, http.MethodPost, base+"/clusters?clusterId=cluster-1", []byte(`{"cluster":{"name":"projects/stackyard/locations/us-central1/clusters/cluster-1"}}`), "createCluster.cluster-1")
	assertGCPRedisClusterSuccess(t, ts, http.MethodPatch, cluster+"?updateMask=size_gb,replica_count", []byte(`{"cluster":{"name":"projects/stackyard/locations/us-central1/clusters/cluster-1","sizeGb":16,"replicaCount":2}}`), "updateCluster.cluster-1")
	assertGCPRedisClusterSuccess(t, ts, http.MethodDelete, cluster, nil, "deleteCluster.cluster-1")

	assertGCPRedisClusterSuccess(t, ts, http.MethodGet, cluster+"/certificateAuthority", nil, "certificateAuthority")
	assertGCPRedisClusterSuccess(t, ts, http.MethodPost, cluster+":rescheduleClusterMaintenance", []byte(`{"name":"projects/stackyard/locations/us-central1/clusters/cluster-1","rescheduleType":"SPECIFIC_TIME","scheduleTime":"2026-01-02T00:00:00Z"}`), "rescheduleClusterMaintenance.cluster-1")

	assertGCPRedisClusterSuccess(t, ts, http.MethodGet, base+"/backupCollections?pageSize=1", nil, "backupCollections")
	assertGCPRedisClusterSuccess(t, ts, http.MethodGet, backupCollection, nil, "backupCollections/collection-1")
	assertGCPRedisClusterSuccess(t, ts, http.MethodGet, backupCollection+"/backups?pageSize=1", nil, "backups")
	assertGCPRedisClusterSuccess(t, ts, http.MethodGet, backup, nil, "backups/backup-1")
	assertGCPRedisClusterSuccess(t, ts, http.MethodDelete, backup, nil, "deleteBackup.backup-1")
	assertGCPRedisClusterSuccess(t, ts, http.MethodPost, backup+":export", []byte(`{"name":"projects/stackyard/locations/us-central1/backupCollections/collection-1/backups/backup-1","gcsBucket":"gs://stackyard-backups/export"}`), "exportBackup.backup-1")
	assertGCPRedisClusterSuccess(t, ts, http.MethodPost, cluster+":backup", []byte(`{"name":"projects/stackyard/locations/us-central1/clusters/cluster-1","backupId":"backup-1"}`), "backupCluster.cluster-1")

	assertGCPRedisClusterSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPRedisClusterSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "locations/us-central1")

	assertGCPRedisClusterSuccess(t, ts, http.MethodGet, base+"/operations?pageSize=1", nil, "operations")
	assertGCPRedisClusterSuccess(t, ts, http.MethodGet, operation, nil, "operations/op-1")
	assertGCPRedisClusterSuccess(t, ts, http.MethodPost, operation+":cancel", []byte(`{}`), "{}")
	assertGCPRedisClusterSuccess(t, ts, http.MethodDelete, operation, nil, "{}")
}

func TestGCPRedisClusterRouter_ListClustersInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisClusterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "redis_cluster",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis_cluster router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisClusterRouter_CreateClusterRequiresClusterID(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisClusterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters", []byte(`{"cluster":{}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "redis_cluster",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis_cluster router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisClusterRouter_UpdateClusterRequiresSupportedUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisClusterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-1?updateMask=nodeType", []byte(`{"cluster":{"name":"projects/stackyard/locations/us-central1/clusters/cluster-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "redis_cluster",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis_cluster router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisClusterRouter_RescheduleSpecificTimeRequiresScheduleTime(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisClusterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-1:rescheduleClusterMaintenance", []byte(`{"rescheduleType":"SPECIFIC_TIME"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "redis_cluster",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis_cluster router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisClusterRouter_RescheduleLockedClusterImmediateFailsPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisClusterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-locked:rescheduleClusterMaintenance", []byte(`{"rescheduleType":"IMMEDIATE"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "redis_cluster",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis_cluster router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisClusterRouter_ExportBackupRequiresGCSBucket(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisClusterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/backupCollections/collection-1/backups/backup-1:export", []byte(`{"name":"projects/stackyard/locations/us-central1/backupCollections/collection-1/backups/backup-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "redis_cluster",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis_cluster router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisClusterRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/redis_cluster?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp redis_cluster contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "redis_cluster" {
		t.Fatalf("expected service=redis_cluster, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func TestGCPRedisClusterRouter_ExportPathParsing(t *testing.T) {
	t.Parallel()

	path := "/gcp/v1/projects/stackyard/locations/us-central1/backupCollections/collection-1/backups/backup-1:export"
	if !isGCPRedisClusterPath(path, true) {
		t.Fatalf("expected export path to be recognized as redis_cluster path")
	}
	if _, _, _, _, ok := parseGCPRedisClusterBackupActionPath(path, "export"); !ok {
		t.Fatalf("expected export path to parse for redis_cluster backup export action")
	}
}

func TestGCPRedisClusterRouter_ExportHandledDirectly(t *testing.T) {
	t.Parallel()

	s := New(Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	req := httptest.NewRequest(http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/backupCollections/collection-1/backups/backup-1:export", bytes.NewReader([]byte(`{"name":"projects/stackyard/locations/us-central1/backupCollections/collection-1/backups/backup-1"}`)))
	req.Header.Set("X-Stackyard-GCP-Service", "redis_cluster")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	if handled := s.handleGCPRedisClusterRouter(rr, req); !handled {
		t.Fatalf("expected export request to be handled by redis_cluster router")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 from direct redis_cluster export handler, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func newGCPRedisClusterContractServer(t *testing.T) *httptest.Server {
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

func assertGCPRedisClusterSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "redis_cluster",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp redis_cluster router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
