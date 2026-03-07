package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPTPURouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPTPUContractServer(t)

	base := "/gcp/v1/projects/stackyard/locations/us-central1"
	node := base + "/nodes/node-1"
	stoppedNode := base + "/nodes/node-stopped"
	operation := base + "/operations/createNode.node-1"

	assertGCPTPUSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, `"locations":[`)
	assertGCPTPUSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, `"locationId":"us-central1"`)

	assertGCPTPUSuccess(t, ts, http.MethodGet, base+"/nodes?pageSize=1", nil, `"nodes":[`)
	assertGCPTPUSuccess(t, ts, http.MethodGet, node, nil, "node-1")
	assertGCPTPUSuccess(t, ts, http.MethodPost, base+"/nodes?nodeId=node-1", []byte(`{
		"node":{
			"name":"projects/stackyard/locations/us-central1/nodes/node-1",
			"acceleratorType":"v3-8",
			"tensorflowVersion":"v2-alpha"
		}
	}`), "operations/createNode.node-1")
	assertGCPTPUSuccess(t, ts, http.MethodDelete, node, nil, "operations/deleteNode.node-1")
	assertGCPTPUSuccess(t, ts, http.MethodPost, node+":reimage", []byte(`{
		"name":"projects/stackyard/locations/us-central1/nodes/node-1",
		"tensorflowVersion":"v2-beta"
	}`), "operations/reimageNode.node-1")
	assertGCPTPUSuccess(t, ts, http.MethodPost, stoppedNode+":start", []byte(`{
		"name":"projects/stackyard/locations/us-central1/nodes/node-stopped"
	}`), "operations/startNode.node-stopped")
	assertGCPTPUSuccess(t, ts, http.MethodPost, node+":stop", []byte(`{
		"name":"projects/stackyard/locations/us-central1/nodes/node-1"
	}`), "operations/stopNode.node-1")

	assertGCPTPUSuccess(t, ts, http.MethodGet, base+"/tensorflowVersions?pageSize=1", nil, `"tensorflowVersions":[`)
	assertGCPTPUSuccess(t, ts, http.MethodGet, base+"/tensorflowVersions/v2-alpha", nil, `"version":"v2-alpha"`)
	assertGCPTPUSuccess(t, ts, http.MethodGet, base+"/acceleratorTypes?pageSize=1", nil, `"acceleratorTypes":[`)
	assertGCPTPUSuccess(t, ts, http.MethodGet, base+"/acceleratorTypes/v3-8", nil, `"type":"v3-8"`)

	assertGCPTPUSuccess(t, ts, http.MethodGet, base+"/operations?pageSize=1", nil, `"operations":[`)
	assertGCPTPUSuccess(t, ts, http.MethodGet, operation, nil, `"done":true`)
	assertGCPTPUSuccess(t, ts, http.MethodPost, operation+":cancel", []byte(`{}`), `{}`)
	assertGCPTPUSuccess(t, ts, http.MethodDelete, operation, nil, `{}`)
}

func TestGCPTPURouter_ListNodesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPTPUContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/nodes?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "tpu",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp tpu list nodes, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTPURouter_CreateNodeRequiresNodeID(t *testing.T) {
	t.Parallel()

	ts := newGCPTPUContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/nodes", []byte(`{
		"node":{"acceleratorType":"v3-8","tensorflowVersion":"v2-alpha"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "tpu",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp tpu create node, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTPURouter_CreateNodeRejectsInvalidNodeID(t *testing.T) {
	t.Parallel()

	ts := newGCPTPUContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/nodes?nodeId=INVALID_NODE", []byte(`{
		"node":{"acceleratorType":"v3-8","tensorflowVersion":"v2-alpha"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "tpu",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp tpu create node invalid nodeId, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTPURouter_CreateNodeNameMustMatchParentAndNodeID(t *testing.T) {
	t.Parallel()

	ts := newGCPTPUContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/nodes?nodeId=node-1", []byte(`{
		"node":{
			"name":"projects/stackyard/locations/us-central1/nodes/node-2",
			"acceleratorType":"v3-8",
			"tensorflowVersion":"v2-alpha"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "tpu",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp tpu create node name mismatch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTPURouter_StartNodeRequiresStoppedState(t *testing.T) {
	t.Parallel()

	ts := newGCPTPUContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/nodes/node-1:start", []byte(`{
		"name":"projects/stackyard/locations/us-central1/nodes/node-1"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "tpu",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp tpu start precondition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTPURouter_StopNodeRequiresReadyState(t *testing.T) {
	t.Parallel()

	ts := newGCPTPUContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/nodes/node-stopped:stop", []byte(`{
		"name":"projects/stackyard/locations/us-central1/nodes/node-stopped"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "tpu",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp tpu stop precondition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTPURouter_ReimageRejectsInvalidTensorFlowVersion(t *testing.T) {
	t.Parallel()

	ts := newGCPTPUContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/nodes/node-1:reimage", []byte(`{
		"name":"projects/stackyard/locations/us-central1/nodes/node-1",
		"tensorflowVersion":"bad version"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "tpu",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp tpu reimage invalid version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTPURouter_GetMissingNodeReturnsNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPTPUContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/nodes/missing-node", nil, map[string]string{
		"X-Stackyard-GCP-Service": "tpu",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp tpu get node missing, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTPURouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPTPUContractServer(t)

	nodeResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/nodes/node-1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "tpu",
	})
	if nodeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp tpu get node, got %d body=%s", nodeResp.StatusCode, string(providerContractBody(t, nodeResp)))
	}
	nodeBody := providerContractJSONMap(t, nodeResp)
	if _, ok := nodeBody["name"].(string); !ok {
		t.Fatalf("expected node.name string, got %#v", nodeBody["name"])
	}
	if _, ok := nodeBody["state"].(string); !ok {
		t.Fatalf("expected node.state string, got %#v", nodeBody["state"])
	}
	if _, ok := nodeBody["health"].(string); !ok {
		t.Fatalf("expected node.health string, got %#v", nodeBody["health"])
	}
	networkEndpoints, _ := nodeBody["networkEndpoints"].([]any)
	if len(networkEndpoints) == 0 {
		t.Fatalf("expected node.networkEndpoints array")
	}
	networkEndpoint, _ := networkEndpoints[0].(map[string]any)
	if _, ok := networkEndpoint["ipAddress"].(string); !ok {
		t.Fatalf("expected networkEndpoints[0].ipAddress string, got %#v", networkEndpoint["ipAddress"])
	}
	if _, ok := networkEndpoint["port"].(float64); !ok {
		t.Fatalf("expected networkEndpoints[0].port number, got %#v", networkEndpoint["port"])
	}

	tfResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/tensorflowVersions?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "tpu",
	})
	if tfResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp tpu list tensorflow versions, got %d body=%s", tfResp.StatusCode, string(providerContractBody(t, tfResp)))
	}
	tfBody := providerContractJSONMap(t, tfResp)
	tfVersions, _ := tfBody["tensorflowVersions"].([]any)
	if len(tfVersions) == 0 {
		t.Fatalf("expected tensorflowVersions array")
	}
	tfVersion, _ := tfVersions[0].(map[string]any)
	if _, ok := tfVersion["name"].(string); !ok {
		t.Fatalf("expected tensorflowVersions[0].name string, got %#v", tfVersion["name"])
	}
	if _, ok := tfVersion["version"].(string); !ok {
		t.Fatalf("expected tensorflowVersions[0].version string, got %#v", tfVersion["version"])
	}

	accelResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/acceleratorTypes?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "tpu",
	})
	if accelResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp tpu list accelerator types, got %d body=%s", accelResp.StatusCode, string(providerContractBody(t, accelResp)))
	}
	accelBody := providerContractJSONMap(t, accelResp)
	acceleratorTypes, _ := accelBody["acceleratorTypes"].([]any)
	if len(acceleratorTypes) == 0 {
		t.Fatalf("expected acceleratorTypes array")
	}
	accelType, _ := acceleratorTypes[0].(map[string]any)
	if _, ok := accelType["name"].(string); !ok {
		t.Fatalf("expected acceleratorTypes[0].name string, got %#v", accelType["name"])
	}
	if _, ok := accelType["type"].(string); !ok {
		t.Fatalf("expected acceleratorTypes[0].type string, got %#v", accelType["type"])
	}

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/nodes?nodeId=node-1", []byte(`{
		"node":{
			"name":"projects/stackyard/locations/us-central1/nodes/node-1",
			"acceleratorType":"v3-8",
			"tensorflowVersion":"v2-alpha"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "tpu",
	})
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp tpu create node, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", createBody["name"])
	}
	if _, ok := createBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", createBody["done"])
	}
	metadata, _ := createBody["metadata"].(map[string]any)
	if _, ok := metadata["target"].(string); !ok {
		t.Fatalf("expected operation metadata.target string, got %#v", metadata["target"])
	}
	if _, ok := metadata["verb"].(string); !ok {
		t.Fatalf("expected operation metadata.verb string, got %#v", metadata["verb"])
	}
	response, _ := createBody["response"].(map[string]any)
	if _, ok := response["name"].(string); !ok {
		t.Fatalf("expected operation response.name string, got %#v", response["name"])
	}

	opsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "tpu",
	})
	if opsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp tpu list operations, got %d body=%s", opsResp.StatusCode, string(providerContractBody(t, opsResp)))
	}
	opsBody := providerContractJSONMap(t, opsResp)
	operations, _ := opsBody["operations"].([]any)
	if len(operations) == 0 {
		t.Fatalf("expected operations array")
	}
	firstOp, _ := operations[0].(map[string]any)
	if _, ok := firstOp["name"].(string); !ok {
		t.Fatalf("expected operation.name string, got %#v", firstOp["name"])
	}
	if _, ok := opsBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", opsBody["nextPageToken"])
	}
}

func TestGCPTPURouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/tpu?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp tpu contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "tpu" {
		t.Fatalf("expected service=tpu, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, got)
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPTPUContractServer(t *testing.T) *httptest.Server {
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

func assertGCPTPUSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "tpu",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp tpu router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
