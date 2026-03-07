package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPTelcoAutomationRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPTelcoAutomationContractServer(t)

	location := "/gcp/v1/projects/stackyard/locations/us-central1"
	clusterName := location + "/orchestrationClusters/cluster-1"
	edgeSlmName := location + "/edgeSlms/edgeslm-1"
	blueprintName := clusterName + "/blueprints/blueprint-draft"
	deploymentName := clusterName + "/deployments/deployment-draft"
	hydratedName := deploymentName + "/hydratedDeployments/hydrated-draft"

	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, location, nil, "us-central1")

	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, location+"/orchestrationClusters?pageSize=1", nil, "orchestrationClusters")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, clusterName, nil, "cluster-1")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, location+"/orchestrationClusters?orchestrationClusterId=cluster-1", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1"}`), "operations/createOrchestrationCluster.cluster-1")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodDelete, clusterName, nil, "operations/deleteOrchestrationCluster.cluster-1")

	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, location+"/edgeSlms?pageSize=1", nil, "edgeSlms")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, edgeSlmName, nil, "edgeslm-1")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, location+"/edgeSlms?edgeSlmId=edgeslm-1", []byte(`{"name":"projects/stackyard/locations/us-central1/edgeSlms/edgeslm-1","orchestrationCluster":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1"}`), "operations/createEdgeSlm.edgeslm-1")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodDelete, edgeSlmName, nil, "operations/deleteEdgeSlm.edgeslm-1")

	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, clusterName+"/blueprints?blueprintId=blueprint-draft", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-draft","sourceBlueprint":"projects/stackyard/locations/us-central1/publicBlueprints/public-blueprint-1","displayName":"Blueprint Draft"}`), "blueprint-draft")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPatch, blueprintName+"?updateMask=display_name", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-draft","sourceBlueprint":"projects/stackyard/locations/us-central1/publicBlueprints/public-blueprint-1","displayName":"Blueprint Draft Updated"}`), "Blueprint Draft Updated")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, blueprintName, nil, "blueprint-draft")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, clusterName+"/blueprints?pageSize=1", nil, "blueprints")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, clusterName+"/blueprints/blueprint-draft:propose", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-draft"}`), "approvalState")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, clusterName+"/blueprints/blueprint-proposed:approve", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-proposed"}`), "approvalState")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, clusterName+"/blueprints/blueprint-proposed:reject", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-proposed"}`), "approvalState")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, clusterName+"/blueprints/blueprint-draft:listRevisions?pageSize=1", nil, "blueprints")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, clusterName+"/blueprints:searchRevisions?query=latest=true&pageSize=1", nil, "blueprints")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, clusterName+"/blueprints/blueprint-draft:discard", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-draft"}`), "{}")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodDelete, blueprintName, nil, "{}")

	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, location+"/publicBlueprints?pageSize=1", nil, "publicBlueprints")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, location+"/publicBlueprints/public-blueprint-1", nil, "public-blueprint-1")

	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, clusterName+"/deployments?deploymentId=deployment-draft", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/deployment-draft","sourceBlueprintRevision":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-approved@rev-3","displayName":"Deployment Draft"}`), "deployment-draft")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPatch, deploymentName+"?updateMask=display_name,source_blueprint_revision", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/deployment-draft","sourceBlueprintRevision":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-approved@rev-3","displayName":"Deployment Draft Updated"}`), "Deployment Draft Updated")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, deploymentName, nil, "deployment-draft")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, clusterName+"/deployments?pageSize=1", nil, "deployments")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, clusterName+"/deployments/deployment-draft:listRevisions?pageSize=1", nil, "deployments")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, clusterName+"/deployments:searchRevisions?query=latest=true&pageSize=1", nil, "deployments")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, deploymentName+":discard", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/deployment-draft"}`), "{}")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, deploymentName+":apply", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/deployment-draft"}`), "rev-applied-1")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, deploymentName+":computeDeploymentStatus", nil, "resourceStatuses")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, clusterName+"/deployments/deployment-applied:rollback", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/deployment-applied","revisionId":"rev-1"}`), "deployment-applied")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, deploymentName+":remove", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/deployment-draft"}`), "{}")

	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, hydratedName, nil, "hydrated-draft")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, deploymentName+"/hydratedDeployments?pageSize=1", nil, "hydratedDeployments")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPatch, hydratedName+"?updateMask=files", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/deployment-draft/hydratedDeployments/hydrated-draft","files":[{"path":"hydrated/site.yaml","content":"kind: ConfigMap","editable":true}]}`), "hydrated-draft")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodPost, hydratedName+":apply", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/deployment-draft/hydratedDeployments/hydrated-draft"}`), "hydrated-draft")

	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, location+"/operations?pageSize=1", nil, "operations")
	assertGCPTelcoAutomationSuccess(t, ts, http.MethodGet, location+"/operations/createOrchestrationCluster.cluster-1", nil, "createOrchestrationCluster.cluster-1")
}

func TestGCPTelcoAutomationRouter_ListOrchestrationClustersInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPTelcoAutomationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "telcoautomation",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp telcoautomation list orchestration clusters, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTelcoAutomationRouter_CreateOrchestrationClusterRequiresID(t *testing.T) {
	t.Parallel()

	ts := newGCPTelcoAutomationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "telcoautomation",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp telcoautomation create orchestration cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTelcoAutomationRouter_CreateEdgeSlmRequiresOrchestrationCluster(t *testing.T) {
	t.Parallel()

	ts := newGCPTelcoAutomationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/edgeSlms?edgeSlmId=edgeslm-1", []byte(`{"name":"projects/stackyard/locations/us-central1/edgeSlms/edgeslm-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "telcoautomation",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp telcoautomation create edge slm, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTelcoAutomationRouter_UpdateBlueprintRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPTelcoAutomationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-draft", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-draft","sourceBlueprint":"projects/stackyard/locations/us-central1/publicBlueprints/public-blueprint-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "telcoautomation",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp telcoautomation update blueprint, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTelcoAutomationRouter_ApproveBlueprintRequiresProposedState(t *testing.T) {
	t.Parallel()

	ts := newGCPTelcoAutomationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-draft:approve", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-draft"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "telcoautomation",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp telcoautomation approve blueprint, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTelcoAutomationRouter_RollbackDeploymentRequiresRevisionID(t *testing.T) {
	t.Parallel()

	ts := newGCPTelcoAutomationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/deployment-applied:rollback", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/deployment-applied"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "telcoautomation",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp telcoautomation rollback deployment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTelcoAutomationRouter_SearchBlueprintRevisionsQueryValidation(t *testing.T) {
	t.Parallel()

	ts := newGCPTelcoAutomationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints:searchRevisions?query=invalid+query", nil, map[string]string{
		"X-Stackyard-GCP-Service": "telcoautomation",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp telcoautomation search blueprint revisions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTelcoAutomationRouter_GetOperationNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPTelcoAutomationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/missing-operation", nil, map[string]string{
		"X-Stackyard-GCP-Service": "telcoautomation",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp telcoautomation get operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTelcoAutomationRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPTelcoAutomationContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "telcoautomation",
	}

	clusterResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1", nil, headers)
	if clusterResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp telcoautomation get orchestration cluster, got %d body=%s", clusterResp.StatusCode, string(providerContractBody(t, clusterResp)))
	}
	clusterBody := providerContractJSONMap(t, clusterResp)
	if _, ok := clusterBody["name"].(string); !ok {
		t.Fatalf("expected orchestration cluster name string, got %#v", clusterBody["name"])
	}
	if _, ok := clusterBody["managementConfig"].(map[string]any); !ok {
		t.Fatalf("expected orchestration cluster managementConfig object, got %#v", clusterBody["managementConfig"])
	}
	if _, ok := clusterBody["state"].(float64); !ok {
		t.Fatalf("expected orchestration cluster state number, got %#v", clusterBody["state"])
	}

	listClustersResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters?pageSize=1", nil, headers)
	if listClustersResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp telcoautomation list orchestration clusters, got %d body=%s", listClustersResp.StatusCode, string(providerContractBody(t, listClustersResp)))
	}
	listClustersBody := providerContractJSONMap(t, listClustersResp)
	if _, ok := listClustersBody["orchestrationClusters"].([]any); !ok {
		t.Fatalf("expected orchestrationClusters array, got %#v", listClustersBody["orchestrationClusters"])
	}
	if _, ok := listClustersBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", listClustersBody["nextPageToken"])
	}
	if _, ok := listClustersBody["unreachable"].([]any); !ok {
		t.Fatalf("expected unreachable array, got %#v", listClustersBody["unreachable"])
	}

	blueprintResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-draft", nil, headers)
	if blueprintResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp telcoautomation get blueprint, got %d body=%s", blueprintResp.StatusCode, string(providerContractBody(t, blueprintResp)))
	}
	blueprintBody := providerContractJSONMap(t, blueprintResp)
	if _, ok := blueprintBody["sourceBlueprint"].(string); !ok {
		t.Fatalf("expected blueprint sourceBlueprint string, got %#v", blueprintBody["sourceBlueprint"])
	}
	if _, ok := blueprintBody["files"].([]any); !ok {
		t.Fatalf("expected blueprint files array, got %#v", blueprintBody["files"])
	}

	deploymentStatusResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/deployment-draft:computeDeploymentStatus", nil, headers)
	if deploymentStatusResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp telcoautomation compute deployment status, got %d body=%s", deploymentStatusResp.StatusCode, string(providerContractBody(t, deploymentStatusResp)))
	}
	deploymentStatusBody := providerContractJSONMap(t, deploymentStatusResp)
	if _, ok := deploymentStatusBody["aggregatedStatus"].(float64); !ok {
		t.Fatalf("expected aggregatedStatus number, got %#v", deploymentStatusBody["aggregatedStatus"])
	}
	if _, ok := deploymentStatusBody["resourceStatuses"].([]any); !ok {
		t.Fatalf("expected resourceStatuses array, got %#v", deploymentStatusBody["resourceStatuses"])
	}

	operationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/createOrchestrationCluster.cluster-1", nil, headers)
	if operationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp telcoautomation get operation, got %d body=%s", operationResp.StatusCode, string(providerContractBody(t, operationResp)))
	}
	operationBody := providerContractJSONMap(t, operationResp)
	if _, ok := operationBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", operationBody["name"])
	}
	if _, ok := operationBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", operationBody["done"])
	}
	if _, ok := operationBody["metadata"].(map[string]any); !ok {
		t.Fatalf("expected operation metadata object, got %#v", operationBody["metadata"])
	}
}

func TestGCPTelcoAutomationRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/telcoautomation?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp telcoautomation contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "telcoautomation" {
		t.Fatalf("expected service=telcoautomation, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPTelcoAutomationContractServer(t *testing.T) *httptest.Server {
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

func assertGCPTelcoAutomationSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "telcoautomation",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp telcoautomation router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
