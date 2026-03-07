package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPWorkflowExecutionsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowExecutionsContractServer(t)
	parent := "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1"
	executionName := parent + "/executions/execution-1"

	assertGCPWorkflowExecutionsSuccess(t, ts, http.MethodGet, parent+"/executions?pageSize=1&view=FULL&filter=state=\"ACTIVE\"&orderBy=startTime%20desc", nil, `"executions"`)
	assertGCPWorkflowExecutionsSuccess(t, ts, http.MethodGet, executionName+"?view=FULL", nil, "executions/execution-1")
	assertGCPWorkflowExecutionsSuccess(t, ts, http.MethodPost, parent+"/executions", []byte(`{
		"execution": {
			"argument": "{\"input\":\"stackyard\"}",
			"callLogLevel": "LOG_ALL_CALLS",
			"labels": {
				"env": "staged",
				"team": "platform"
			}
		}
	}`), "executions/execution-created")
	assertGCPWorkflowExecutionsSuccess(t, ts, http.MethodPost, executionName+":cancel", []byte(`{}`), `"state":"CANCELLED"`)
}

func TestGCPWorkflowExecutionsRouter_GRPCBridgeRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowExecutionsContractServer(t)
	parent := "projects/stackyard/locations/us-central1/workflows/workflow-1"
	name := parent + "/executions/execution-1"

	assertGCPWorkflowExecutionsGRPCBridgeSuccess(t, ts, "/gcp/google.cloud.workflows.executions.v1.Executions/ListExecutions", []byte(`{
		"parent": "projects/stackyard/locations/us-central1/workflows/workflow-1",
		"pageSize": 1
	}`), `"executions"`)
	assertGCPWorkflowExecutionsGRPCBridgeSuccess(t, ts, "/gcp/google.cloud.workflows.executions.v1.Executions/GetExecution", []byte(`{
		"name": "projects/stackyard/locations/us-central1/workflows/workflow-1/executions/execution-1",
		"view": "FULL"
	}`), name)
	assertGCPWorkflowExecutionsGRPCBridgeSuccess(t, ts, "/gcp/google.cloud.workflows.executions.v1.Executions/CreateExecution", []byte(`{
		"parent": "projects/stackyard/locations/us-central1/workflows/workflow-1",
		"execution": {
			"argument": "{\"input\":\"bridge\"}",
			"labels": {"env": "staged"}
		}
	}`), "executions/execution-created")
	assertGCPWorkflowExecutionsGRPCBridgeSuccess(t, ts, "/gcp/google.cloud.workflows.executions.v1.Executions/CancelExecution", []byte(`{
		"name": "projects/stackyard/locations/us-central1/workflows/workflow-1/executions/execution-1"
	}`), `"state":"CANCELLED"`)
}

func TestGCPWorkflowExecutionsRouter_ListExecutionsRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowExecutionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflow executions list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowExecutionsRouter_ListExecutionsRejectsInvalidView(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowExecutionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions?view=UNKNOWN", nil, map[string]string{
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflow executions list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowExecutionsRouter_ListExecutionsRejectsUnsupportedFilterAndOrderBy(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowExecutionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions?filter=state=ACTIVE", nil, map[string]string{
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflow executions list filter, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions?orderBy=name", nil, map[string]string{
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp workflow executions list orderBy, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowExecutionsRouter_CreateExecutionValidatesInput(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowExecutionsContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing execution payload, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions", []byte(`{
		"execution": {
			"labels": {"BadKey": "value"}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid labels, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions", []byte(`{
		"execution": {
			"argument": "`+strings.Repeat("a", 32769)+`"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized argument, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestGCPWorkflowExecutionsRouter_CancelExecutionValidatesStateAndName(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowExecutionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions/execution-2:cancel", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from terminal state cancel, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions/execution-1:cancel", []byte(`{
		"name": "projects/stackyard/locations/us-central1/workflows/workflow-1/executions/execution-2"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from cancel name mismatch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowExecutionsRouter_GetExecutionNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowExecutionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions/missing-execution", nil, map[string]string{
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from get missing execution, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWorkflowExecutionsRouter_OutputShapeAssertions(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkflowExecutionsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "workflow-executions",
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions?pageSize=1&view=FULL", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from list executions, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	executions, ok := listBody["executions"].([]any)
	if !ok || len(executions) == 0 {
		t.Fatalf("expected executions array, got %#v", listBody["executions"])
	}
	firstExecution, _ := executions[0].(map[string]any)
	if _, ok := firstExecution["name"].(string); !ok {
		t.Fatalf("expected execution.name string, got %#v", firstExecution["name"])
	}
	if _, ok := firstExecution["state"].(string); !ok {
		t.Fatalf("expected execution.state string, got %#v", firstExecution["state"])
	}
	if _, ok := firstExecution["startTime"].(string); !ok {
		t.Fatalf("expected execution.startTime string, got %#v", firstExecution["startTime"])
	}
	if _, ok := firstExecution["duration"].(string); !ok {
		t.Fatalf("expected execution.duration string, got %#v", firstExecution["duration"])
	}
	if _, ok := firstExecution["argument"].(string); !ok {
		t.Fatalf("expected execution.argument string, got %#v", firstExecution["argument"])
	}
	if _, ok := firstExecution["result"].(string); !ok {
		t.Fatalf("expected execution.result string, got %#v", firstExecution["result"])
	}
	if _, ok := firstExecution["error"].(map[string]any); !ok {
		t.Fatalf("expected execution.error object, got %#v", firstExecution["error"])
	}
	if _, ok := firstExecution["status"].(map[string]any); !ok {
		t.Fatalf("expected execution.status object, got %#v", firstExecution["status"])
	}
	if _, ok := firstExecution["labels"].(map[string]any); !ok {
		t.Fatalf("expected execution.labels object, got %#v", firstExecution["labels"])
	}

	getResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflows/workflow-1/executions/execution-1?view=BASIC", nil, headers)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get execution basic view, got %d body=%s", getResp.StatusCode, string(providerContractBody(t, getResp)))
	}
	getBody := providerContractJSONMap(t, getResp)
	if _, exists := getBody["argument"]; exists {
		t.Fatalf("expected execution.argument omitted in BASIC view, got %#v", getBody["argument"])
	}
	if _, exists := getBody["labels"]; exists {
		t.Fatalf("expected execution.labels omitted in BASIC view, got %#v", getBody["labels"])
	}
}

func TestGCPWorkflowExecutionsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workflow-executions?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from workflow executions contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "workflow_executions" {
		t.Fatalf("expected service=workflow_executions, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, got)
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPWorkflowExecutionsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPWorkflowExecutionsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "workflow-executions",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from workflow executions for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPWorkflowExecutionsGRPCBridgeSuccess(t *testing.T, ts *httptest.Server, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, http.MethodPost, path, payload, map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from workflow executions grpc bridge for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected grpc bridge response body for %s: %s", path, body)
	}
}
