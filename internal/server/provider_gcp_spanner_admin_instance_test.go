package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSpannerAdminInstanceRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminInstanceContractServer(t)

	configs := "/gcp/v1/projects/stackyard/instanceConfigs"
	config := configs + "/custom-stackyard-primary"
	instances := "/gcp/v1/projects/stackyard/instances"
	instance := instances + "/stackyard-instance"
	partitions := instance + "/instancePartitions"
	partition := partitions + "/partition-a"
	operation := instance + "/operations/create-instance-stackyard-instance"

	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, configs+"?pageSize=1", nil, "instanceConfigs")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, config, nil, "custom-stackyard-primary")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodPost, configs+"?instanceConfigId=custom-stackyard-new", []byte(`{"parent":"projects/stackyard","instanceConfig":{"displayName":"Stackyard New Config"}}`), "create-instance-config-custom-stackyard-new")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodPatch, config+"?updateMask=displayName", []byte(`{"instanceConfig":{"name":"projects/stackyard/instanceConfigs/custom-stackyard-primary","displayName":"Updated Config"}}`), "update-instance-config-custom-stackyard-primary")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodDelete, config, nil, "{}")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/instanceConfigOperations?pageSize=1", nil, "operations")

	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, instances+"?pageSize=1", nil, "instances")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, instance, nil, "stackyard-instance")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodPost, instances+"?instanceId=stackyard-instance-new", []byte(`{"parent":"projects/stackyard","instance":{"config":"projects/stackyard/instanceConfigs/custom-stackyard-primary","displayName":"New Instance","nodeCount":2}}`), "create-instance-stackyard-instance-new")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodPatch, instance+"?updateMask=displayName", []byte(`{"instance":{"name":"projects/stackyard/instances/stackyard-instance","displayName":"Updated Instance","nodeCount":3}}`), "update-instance-stackyard-instance")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodPost, instance+":move", []byte(`{"targetConfig":"projects/stackyard/instanceConfigs/custom-stackyard-analytics"}`), "move-instance-stackyard-instance")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodDelete, instance, nil, "{}")

	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, partitions+"?pageSize=1", nil, "instancePartitions")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, partition, nil, "partition-a")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodPost, partitions+"?instancePartitionId=partition-new", []byte(`{"parent":"projects/stackyard/instances/stackyard-instance","instancePartition":{"displayName":"New Partition"}}`), "create-instance-partition-partition-new")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodPatch, partition+"?updateMask=displayName", []byte(`{"instancePartition":{"name":"projects/stackyard/instances/stackyard-instance/instancePartitions/partition-a","displayName":"Updated Partition"}}`), "update-instance-partition-partition-a")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, instance+"/instancePartitionOperations?pageSize=1", nil, "operations")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodDelete, partition, nil, "{}")

	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodPost, instance+":setIamPolicy", []byte(`{"policy":{"bindings":[{"role":"roles/spanner.admin","members":["user:stackyard@example.com"]}]}}`), "bindings")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, instance+":getIamPolicy", nil, "roles/spanner.admin")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodPost, instance+":testIamPermissions", []byte(`{"permissions":["spanner.instances.get","resourcemanager.projects.get"]}`), "spanner.instances.get")

	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, operation, nil, "create-instance-stackyard-instance")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, instance+"/operations?pageSize=1", nil, "operations")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/operations?pageSize=1", nil, "operations")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodPost, operation+":cancel", []byte(`{}`), "{}")
	assertGCPSpannerAdminInstanceSuccess(t, ts, http.MethodDelete, operation, nil, "{}")
}

func TestGCPSpannerAdminInstanceRouter_ListInstancesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminInstanceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/instances?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-instance",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner admin instance list instances, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdminInstanceRouter_CreateInstanceRequiresDisplayName(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminInstanceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances?instanceId=stackyard-instance-new", []byte(`{"instance":{"config":"projects/stackyard/instanceConfigs/custom-stackyard-primary"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-admin-instance",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner admin instance create instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdminInstanceRouter_MoveInstanceRequiresDifferentConfig(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminInstanceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance:move", []byte(`{"targetConfig":"projects/stackyard/instanceConfigs/custom-stackyard-primary"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-admin-instance",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner admin instance move instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdminInstanceRouter_GetInstanceNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminInstanceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/instances/missing-instance", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-instance",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp spanner admin instance get instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdminInstanceRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminInstanceContractServer(t)

	listConfigsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/instanceConfigs?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-instance",
	})
	if listConfigsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from list instance configs, got %d body=%s", listConfigsResp.StatusCode, string(providerContractBody(t, listConfigsResp)))
	}
	listConfigsBody := providerContractJSONMap(t, listConfigsResp)
	configs, ok := listConfigsBody["instanceConfigs"].([]any)
	if !ok || len(configs) == 0 {
		t.Fatalf("expected instanceConfigs array, got %#v", listConfigsBody["instanceConfigs"])
	}
	firstConfig, _ := configs[0].(map[string]any)
	if _, ok := firstConfig["name"].(string); !ok {
		t.Fatalf("expected instanceConfigs[0].name string, got %#v", firstConfig["name"])
	}
	if _, ok := firstConfig["reconciling"].(bool); !ok {
		t.Fatalf("expected instanceConfigs[0].reconciling bool, got %#v", firstConfig["reconciling"])
	}

	getInstanceResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/instances/stackyard-instance", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-instance",
	})
	if getInstanceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get instance, got %d body=%s", getInstanceResp.StatusCode, string(providerContractBody(t, getInstanceResp)))
	}
	getInstanceBody := providerContractJSONMap(t, getInstanceResp)
	if _, ok := getInstanceBody["name"].(string); !ok {
		t.Fatalf("expected instance.name string, got %#v", getInstanceBody["name"])
	}
	if _, ok := getInstanceBody["nodeCount"].(float64); !ok {
		t.Fatalf("expected instance.nodeCount numeric, got %#v", getInstanceBody["nodeCount"])
	}
	if _, ok := getInstanceBody["endpointUris"].([]any); !ok {
		t.Fatalf("expected instance.endpointUris array, got %#v", getInstanceBody["endpointUris"])
	}

	getPartitionResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/instances/stackyard-instance/instancePartitions/partition-a", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-instance",
	})
	if getPartitionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get instance partition, got %d body=%s", getPartitionResp.StatusCode, string(providerContractBody(t, getPartitionResp)))
	}
	getPartitionBody := providerContractJSONMap(t, getPartitionResp)
	if _, ok := getPartitionBody["name"].(string); !ok {
		t.Fatalf("expected instancePartition.name string, got %#v", getPartitionBody["name"])
	}
	if _, ok := getPartitionBody["processingUnits"].(float64); !ok {
		t.Fatalf("expected instancePartition.processingUnits numeric, got %#v", getPartitionBody["processingUnits"])
	}

	getOperationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/instances/stackyard-instance/operations/create-instance-stackyard-instance", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-instance",
	})
	if getOperationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get operation, got %d body=%s", getOperationResp.StatusCode, string(providerContractBody(t, getOperationResp)))
	}
	getOperationBody := providerContractJSONMap(t, getOperationResp)
	if _, ok := getOperationBody["name"].(string); !ok {
		t.Fatalf("expected operation.name string, got %#v", getOperationBody["name"])
	}
	if _, ok := getOperationBody["done"].(bool); !ok {
		t.Fatalf("expected operation.done bool, got %#v", getOperationBody["done"])
	}

	permissionsResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance:testIamPermissions", []byte(`{"permissions":["spanner.instances.get","resourcemanager.projects.get"]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-admin-instance",
	})
	if permissionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from test iam permissions, got %d body=%s", permissionsResp.StatusCode, string(providerContractBody(t, permissionsResp)))
	}
	permissionsBody := providerContractJSONMap(t, permissionsResp)
	permissions, ok := permissionsBody["permissions"].([]any)
	if !ok || len(permissions) != 1 {
		t.Fatalf("expected filtered permissions length 1, got %#v", permissionsBody["permissions"])
	}
}

func TestGCPSpannerAdminInstanceRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/spanner_admin_instance?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp spanner admin instance contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "spanner_admin_instance" {
		t.Fatalf("expected service=spanner_admin_instance, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPSpannerAdminInstanceContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSpannerAdminInstanceSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-instance",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp spanner admin instance router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
