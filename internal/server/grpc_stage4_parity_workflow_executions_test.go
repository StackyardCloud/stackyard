package server

import (
	"net/http"
	"strings"
	"testing"

	executionspb "cloud.google.com/go/workflows/executions/apiv1/executionspb"
)

func TestGCPStage4GRPCParity_WorkflowExecutions(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	parent := "projects/stackyard/locations/us-central1/workflows/workflow-1"
	executionName := parent + "/executions/execution-1"
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "workflow-executions",
	}

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+parent+"/executions?pageSize=1&view=FULL&filter=state=\"ACTIVE\"", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workflow executions list, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restExecutions, ok := restListBody["executions"].([]any)
	if !ok || len(restExecutions) == 0 {
		t.Fatalf("expected executions list in rest payload, got %#v", restListBody["executions"])
	}
	restFirstExecution, _ := restExecutions[0].(map[string]any)
	restFirstExecutionName, _ := restFirstExecution["name"].(string)

	var grpcListResp executionspb.ListExecutionsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpWorkflowExecutionsListExecutionsMethod, &executionspb.ListExecutionsRequest{
		Parent:   parent,
		PageSize: 1,
		View:     executionspb.ExecutionView_FULL,
		Filter:   `state="ACTIVE"`,
	}, &grpcListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for workflow executions list, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcListResp.GetExecutions()) != 1 {
		t.Fatalf("expected one grpc execution, got %d", len(grpcListResp.GetExecutions()))
	}
	if grpcListResp.GetExecutions()[0].GetName() != restFirstExecutionName {
		t.Fatalf("expected grpc execution name %q to match rest %q", grpcListResp.GetExecutions()[0].GetName(), restFirstExecutionName)
	}

	restGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+executionName+"?view=FULL", nil, headers)
	if restGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workflow executions get, got %d body=%s", restGetResp.StatusCode, string(providerContractBody(t, restGetResp)))
	}
	restGetBody := providerContractJSONMap(t, restGetResp)
	restState, _ := restGetBody["state"].(string)

	var grpcGetResp executionspb.Execution
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowExecutionsGetExecutionMethod, &executionspb.GetExecutionRequest{
		Name: executionName,
		View: executionspb.ExecutionView_FULL,
	}, &grpcGetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for workflow executions get, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcGetResp.GetName() != executionName {
		t.Fatalf("expected grpc execution name %q, got %q", executionName, grpcGetResp.GetName())
	}
	if grpcGetResp.GetState().String() != restState {
		t.Fatalf("expected grpc execution state %q to match rest %q", grpcGetResp.GetState().String(), restState)
	}

	restCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+parent+"/executions", []byte(`{
		"execution": {
			"argument": "{\"input\":\"grpc\"}",
			"labels": {"env": "staged"}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if restCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workflow executions create, got %d body=%s", restCreateResp.StatusCode, string(providerContractBody(t, restCreateResp)))
	}
	restCreateBody := providerContractJSONMap(t, restCreateResp)
	restCreateName, _ := restCreateBody["name"].(string)

	var grpcCreateResp executionspb.Execution
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowExecutionsCreateExecutionMethod, &executionspb.CreateExecutionRequest{
		Parent: parent,
		Execution: &executionspb.Execution{
			Argument: `{"input":"grpc"}`,
			Labels: map[string]string{
				"env": "staged",
			},
		},
	}, &grpcCreateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for workflow executions create, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcCreateResp.GetName() != restCreateName {
		t.Fatalf("expected grpc created execution name %q to match rest %q", grpcCreateResp.GetName(), restCreateName)
	}

	restCancelResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+executionName+":cancel", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflow-executions",
	})
	if restCancelResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workflow executions cancel, got %d body=%s", restCancelResp.StatusCode, string(providerContractBody(t, restCancelResp)))
	}
	restCancelBody := providerContractJSONMap(t, restCancelResp)
	restCancelState, _ := restCancelBody["state"].(string)

	var grpcCancelResp executionspb.Execution
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowExecutionsCancelExecutionMethod, &executionspb.CancelExecutionRequest{
		Name: executionName,
	}, &grpcCancelResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for workflow executions cancel, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcCancelResp.GetState().String() != restCancelState {
		t.Fatalf("expected grpc cancel state %q to match rest %q", grpcCancelResp.GetState().String(), restCancelState)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowExecutionsListExecutionsMethod, &executionspb.ListExecutionsRequest{
		PageSize: 1,
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for workflow executions list missing parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowExecutionsGetExecutionMethod, &executionspb.GetExecutionRequest{
		Name: executionName,
		View: executionspb.ExecutionView(99),
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "view-invalid") {
		t.Fatalf("expected grpc invalid argument for workflow executions get invalid view, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowExecutionsCancelExecutionMethod, &executionspb.CancelExecutionRequest{
		Name: parent + "/executions/execution-2",
	}, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "execution-not-cancellable") {
		t.Fatalf("expected grpc failed precondition for workflow executions cancel terminal state, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowExecutionsGetExecutionMethod, &executionspb.GetExecutionRequest{
		Name: parent + "/executions/missing-execution",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "execution-not-found") {
		t.Fatalf("expected grpc not found for missing execution, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
