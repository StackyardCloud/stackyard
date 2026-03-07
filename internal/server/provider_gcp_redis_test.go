package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPRedisRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisContractServer(t)

	base := "/gcp/v1/projects/stackyard/locations/us-central1"
	instance := base + "/instances/redis-1"
	operation := base + "/operations/op-1"

	assertGCPRedisSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPRedisSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "locations/us-central1")

	assertGCPRedisSuccess(t, ts, http.MethodGet, base+"/instances?pageSize=1", nil, "instances")
	assertGCPRedisSuccess(t, ts, http.MethodGet, instance, nil, "instances/redis-1")
	assertGCPRedisSuccess(t, ts, http.MethodGet, instance+"/authString", nil, "authString")
	assertGCPRedisSuccess(t, ts, http.MethodPost, base+"/instances?instanceId=redis-1", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1"}`), "createInstance.redis-1")
	assertGCPRedisSuccess(t, ts, http.MethodPatch, instance+"?updateMask=display_name,labels", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1","displayName":"Redis One Updated"}`), "updateInstance.redis-1")
	assertGCPRedisSuccess(t, ts, http.MethodDelete, instance, nil, "deleteInstance.redis-1")

	assertGCPRedisSuccess(t, ts, http.MethodPost, instance+":upgrade", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1","redisVersion":"REDIS_7_0"}`), "upgradeInstance.redis-1")
	assertGCPRedisSuccess(t, ts, http.MethodPost, instance+":import", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1","inputConfig":{"gcsSource":{"uri":"gs://stackyard/import.rdb"}}}`), "importInstance.redis-1")
	assertGCPRedisSuccess(t, ts, http.MethodPost, instance+":export", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1","outputConfig":{"gcsDestination":{"uri":"gs://stackyard/export.rdb"}}}`), "exportInstance.redis-1")
	assertGCPRedisSuccess(t, ts, http.MethodPost, instance+":failover", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1"}`), "failoverInstance.redis-1")
	assertGCPRedisSuccess(t, ts, http.MethodPost, instance+":rescheduleMaintenance", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1","rescheduleType":"SPECIFIC_TIME","scheduleTime":"2026-01-02T00:00:00Z"}`), "rescheduleMaintenance.redis-1")

	assertGCPRedisSuccess(t, ts, http.MethodGet, base+"/operations?pageSize=1", nil, "operations")
	assertGCPRedisSuccess(t, ts, http.MethodGet, operation, nil, "operations/op-1")
	assertGCPRedisSuccess(t, ts, http.MethodPost, operation+":cancel", []byte(`{}`), "{}")
	assertGCPRedisSuccess(t, ts, http.MethodDelete, operation, nil, "{}")
}

func TestGCPRedisRouter_ListInstancesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "redis",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisRouter_CreateInstanceRequiresInstanceID(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "redis",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisRouter_UpdateInstanceRequiresSupportedUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/instances/redis-1?updateMask=nodeType", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "redis",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisRouter_UpgradeRequiresRedisVersion(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/redis-1:upgrade", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "redis",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisRouter_ExportRequiresDestinationURI(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/redis-1:export", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1","outputConfig":{}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "redis",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisRouter_RescheduleSpecificTimeRequiresScheduleTime(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/redis-1:rescheduleMaintenance", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-1","rescheduleType":"SPECIFIC_TIME"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "redis",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisRouter_FailoverBasicTierFailsPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPRedisContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/redis-basic:failover", []byte(`{"name":"projects/stackyard/locations/us-central1/instances/redis-basic"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "redis",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp redis router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRedisRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/redis?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp redis contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "redis" {
		t.Fatalf("expected service=redis, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPRedisContractServer(t *testing.T) *httptest.Server {
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

func assertGCPRedisSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "redis",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp redis router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
