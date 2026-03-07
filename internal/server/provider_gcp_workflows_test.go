package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPWorkflowsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowsContractServer(t)

	parentPath := "/gcp/v1/projects/stackyard/locations/us-central1"
	workflowPath := parentPath + "/workflows/workflow-1"
	workflowName := "projects/stackyard/locations/us-central1/workflows/workflow-1"

	assertGCPWorkflowsSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, `"locations"`)
	assertGCPWorkflowsSuccess(t, ts, http.MethodGet, parentPath, nil, "us-central1")
	assertGCPWorkflowsSuccess(t, ts, http.MethodGet, parentPath+"/workflows?pageSize=1&filter=state=\"ACTIVE\"&orderBy=createTime", nil, `"workflows"`)
	assertGCPWorkflowsSuccess(t, ts, http.MethodGet, workflowPath+"?revisionId=000001-a4d", nil, workflowName)
	assertGCPWorkflowsSuccess(t, ts, http.MethodGet, workflowPath+":listRevisions?pageSize=1", nil, `"workflows"`)
	assertGCPWorkflowsSuccess(t, ts, http.MethodPost, parentPath+"/workflows?workflowId=workflow-3", []byte(`{
		"workflow": {
			"name": "projects/stackyard/locations/us-central1/workflows/workflow-3",
			"description": "workflow create",
			"sourceContents": "main:\n  steps:\n  - done:\n      return: \"ok\""
		}
	}`), "operations/createWorkflow.workflow-3")
	assertGCPWorkflowsSuccess(t, ts, http.MethodPatch, workflowPath+"?updateMask=description", []byte(`{
		"workflow": {
			"name": "projects/stackyard/locations/us-central1/workflows/workflow-1",
			"description": "workflow update",
			"sourceContents": "main:\n  steps:\n  - done:\n      return: \"updated\""
		}
	}`), "operations/updateWorkflow.workflow-1")
	assertGCPWorkflowsSuccess(t, ts, http.MethodGet, parentPath+"/operations?pageSize=1&filter=done=false", nil, `"operations"`)
	assertGCPWorkflowsSuccess(t, ts, http.MethodGet, parentPath+"/operations/op-1", nil, "operations/op-1")
	assertGCPWorkflowsSuccess(t, ts, http.MethodDelete, parentPath+"/operations/op-1", nil, "{}")
	assertGCPWorkflowsSuccess(t, ts, http.MethodDelete, workflowPath, nil, "operations/deleteWorkflow.workflow-1")
}

func TestGCPWorkflowsRouter_ListWorkflowsRejectsUnsupportedFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows?filter=state=ACTIVE", nil, map[string]string{
		"X-Stackyard-GCP-Service": "workflows",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflows list workflows, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowsRouter_ListWorkflowsRejectsUnsupportedOrderBy(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows?orderBy=name", nil, map[string]string{
		"X-Stackyard-GCP-Service": "workflows",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflows list workflows, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowsRouter_GetWorkflowRejectsInvalidRevisionID(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1?revisionId=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "workflows",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflows get workflow, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowsRouter_CreateWorkflowRequiresWorkflowID(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/workflows", []byte(`{
		"workflow": {
			"sourceContents": "main:\n  steps:\n  - done:\n      return: \"ok\""
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflows",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflows create workflow, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowsRouter_CreateWorkflowRejectsInvalidWorkflowID(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/workflows?workflowId=1bad", []byte(`{
		"workflow": {
			"sourceContents": "main:\n  steps:\n  - done:\n      return: \"ok\""
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflows",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflows create workflow, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowsRouter_CreateWorkflowRequiresSourceContents(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/workflows?workflowId=workflow-1", []byte(`{
		"workflow": {
			"name": "projects/stackyard/locations/us-central1/workflows/workflow-1"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflows",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflows create workflow, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowsRouter_UpdateWorkflowNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1?updateMask=description", []byte(`{
		"workflow": {
			"name": "projects/stackyard/locations/us-central1/workflows/workflow-2",
			"sourceContents": "main:\n  steps:\n  - done:\n      return: \"ok\""
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflows",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflows update workflow, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowsRouter_ListOperationsRejectsInvalidFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?filter=done=yes", nil, map[string]string{
		"X-Stackyard-GCP-Service": "workflows",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflows list operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowsRouter_OutputShapeAssertions(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "workflows",
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows?pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp workflows list, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	workflows, ok := listBody["workflows"].([]any)
	if !ok || len(workflows) == 0 {
		t.Fatalf("expected workflows array, got %#v", listBody["workflows"])
	}
	firstWorkflow, _ := workflows[0].(map[string]any)
	if _, ok := firstWorkflow["name"].(string); !ok {
		t.Fatalf("expected workflow.name string, got %#v", firstWorkflow["name"])
	}
	if _, ok := firstWorkflow["revisionId"].(string); !ok {
		t.Fatalf("expected workflow.revisionId string, got %#v", firstWorkflow["revisionId"])
	}
	if _, ok := firstWorkflow["state"].(string); !ok {
		t.Fatalf("expected workflow.state string, got %#v", firstWorkflow["state"])
	}
	if _, ok := firstWorkflow["sourceContents"].(string); !ok {
		t.Fatalf("expected workflow.sourceContents string, got %#v", firstWorkflow["sourceContents"])
	}
	if _, ok := firstWorkflow["labels"].(map[string]any); !ok {
		t.Fatalf("expected workflow.labels object, got %#v", firstWorkflow["labels"])
	}

	locationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, headers)
	if locationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp workflows get location, got %d body=%s", locationResp.StatusCode, string(providerContractBody(t, locationResp)))
	}
	locationBody := providerContractJSONMap(t, locationResp)
	if _, ok := locationBody["name"].(string); !ok {
		t.Fatalf("expected location.name string, got %#v", locationBody["name"])
	}
	if _, ok := locationBody["locationId"].(string); !ok {
		t.Fatalf("expected location.locationId string, got %#v", locationBody["locationId"])
	}
	if _, ok := locationBody["metadata"].(map[string]any); !ok {
		t.Fatalf("expected location.metadata object, got %#v", locationBody["metadata"])
	}

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/workflows?workflowId=workflow-typed", []byte(`{
		"workflow": {
			"name": "projects/stackyard/locations/us-central1/workflows/workflow-typed",
			"sourceContents": "main:\n  steps:\n  - done:\n      return: \"typed\""
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflows",
	})
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp workflows create workflow, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected operation.name string, got %#v", createBody["name"])
	}
	if _, ok := createBody["done"].(bool); !ok {
		t.Fatalf("expected operation.done bool, got %#v", createBody["done"])
	}
	metadata, ok := createBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation.metadata object, got %#v", createBody["metadata"])
	}
	if _, ok := metadata["target"].(string); !ok {
		t.Fatalf("expected operation.metadata.target string, got %#v", metadata["target"])
	}

	getOperationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-2", nil, headers)
	if getOperationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp workflows get operation, got %d body=%s", getOperationResp.StatusCode, string(providerContractBody(t, getOperationResp)))
	}
	getOperationBody := providerContractJSONMap(t, getOperationResp)
	if _, ok := getOperationBody["name"].(string); !ok {
		t.Fatalf("expected operation.name string, got %#v", getOperationBody["name"])
	}
	if _, ok := getOperationBody["done"].(bool); !ok {
		t.Fatalf("expected operation.done bool, got %#v", getOperationBody["done"])
	}
	if _, ok := getOperationBody["metadata"].(map[string]any); !ok {
		t.Fatalf("expected operation.metadata object, got %#v", getOperationBody["metadata"])
	}
}

func TestGCPWorkflowsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp workflows contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "workflows" {
		t.Fatalf("expected service=workflows, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPWorkflowsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPWorkflowsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "workflows",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp workflows router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
