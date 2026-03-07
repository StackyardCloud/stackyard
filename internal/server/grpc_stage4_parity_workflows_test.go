package server

import (
	"net/http"
	"strings"
	"testing"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	workflowspb "cloud.google.com/go/workflows/apiv1/workflowspb"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestGCPStage4GRPCParity_Workflows(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	parent := "projects/stackyard/locations/us-central1"
	workflowName := parent + "/workflows/workflow-1"
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "workflows",
	}

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+parent+"/workflows?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workflows list, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restWorkflows, ok := restListBody["workflows"].([]any)
	if !ok || len(restWorkflows) == 0 {
		t.Fatalf("expected workflows list in rest payload, got %#v", restListBody["workflows"])
	}
	restFirstWorkflow, _ := restWorkflows[0].(map[string]any)
	restFirstWorkflowName, _ := restFirstWorkflow["name"].(string)

	var grpcListResp workflowspb.ListWorkflowsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpWorkflowsListWorkflowsMethod, &workflowspb.ListWorkflowsRequest{
		Parent:   parent,
		PageSize: 1,
	}, &grpcListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for workflows list, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcListResp.GetWorkflows()) != 1 {
		t.Fatalf("expected one grpc workflow, got %d", len(grpcListResp.GetWorkflows()))
	}
	if grpcListResp.GetWorkflows()[0].GetName() != restFirstWorkflowName {
		t.Fatalf("expected grpc workflow name %q to match rest %q", grpcListResp.GetWorkflows()[0].GetName(), restFirstWorkflowName)
	}

	restGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+workflowName+"?revisionId=000001-a4d", nil, headers)
	if restGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workflows get, got %d body=%s", restGetResp.StatusCode, string(providerContractBody(t, restGetResp)))
	}
	restGetBody := providerContractJSONMap(t, restGetResp)
	restRevisionID, _ := restGetBody["revisionId"].(string)

	var grpcGetResp workflowspb.Workflow
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowsGetWorkflowMethod, &workflowspb.GetWorkflowRequest{
		Name:       workflowName,
		RevisionId: "000001-a4d",
	}, &grpcGetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for workflows get, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcGetResp.GetName() != workflowName {
		t.Fatalf("expected grpc workflow name %q, got %q", workflowName, grpcGetResp.GetName())
	}
	if grpcGetResp.GetRevisionId() != restRevisionID {
		t.Fatalf("expected grpc workflow revision %q to match rest %q", grpcGetResp.GetRevisionId(), restRevisionID)
	}

	restCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+parent+"/workflows?workflowId=workflow-3", []byte(`{
		"workflow": {
			"name": "projects/stackyard/locations/us-central1/workflows/workflow-3",
			"sourceContents": "main:\n  steps:\n  - done:\n      return: \"ok\""
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workflows",
	})
	if restCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workflows create, got %d body=%s", restCreateResp.StatusCode, string(providerContractBody(t, restCreateResp)))
	}
	restCreateBody := providerContractJSONMap(t, restCreateResp)
	restCreateOpName, _ := restCreateBody["name"].(string)

	var grpcCreateResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowsCreateWorkflowMethod, &workflowspb.CreateWorkflowRequest{
		Parent:     parent,
		WorkflowId: "workflow-3",
		Workflow: &workflowspb.Workflow{
			Name: workflowName[:len(workflowName)-len("workflow-1")] + "workflow-3",
			SourceCode: &workflowspb.Workflow_SourceContents{
				SourceContents: "main:\n  steps:\n  - done:\n      return: \"ok\"",
			},
		},
	}, &grpcCreateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for workflows create, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcCreateResp.GetName() != restCreateOpName {
		t.Fatalf("expected grpc operation name %q to match rest %q", grpcCreateResp.GetName(), restCreateOpName)
	}

	restRevisionsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+workflowName+":listRevisions?pageSize=1", nil, headers)
	if restRevisionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workflows list revisions, got %d body=%s", restRevisionsResp.StatusCode, string(providerContractBody(t, restRevisionsResp)))
	}
	restRevisionsBody := providerContractJSONMap(t, restRevisionsResp)
	restRevisionItems, ok := restRevisionsBody["workflows"].([]any)
	if !ok || len(restRevisionItems) == 0 {
		t.Fatalf("expected revisions list in rest payload, got %#v", restRevisionsBody["workflows"])
	}
	restFirstRevision, _ := restRevisionItems[0].(map[string]any)
	restFirstRevisionID, _ := restFirstRevision["revisionId"].(string)

	var grpcRevisionsResp workflowspb.ListWorkflowRevisionsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowsListWorkflowRevisionsMethod, &workflowspb.ListWorkflowRevisionsRequest{
		Name:     workflowName,
		PageSize: 1,
	}, &grpcRevisionsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for workflows list revisions, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcRevisionsResp.GetWorkflows()) != 1 {
		t.Fatalf("expected one grpc workflow revision, got %d", len(grpcRevisionsResp.GetWorkflows()))
	}
	if grpcRevisionsResp.GetWorkflows()[0].GetRevisionId() != restFirstRevisionID {
		t.Fatalf("expected grpc revision %q to match rest %q", grpcRevisionsResp.GetWorkflows()[0].GetRevisionId(), restFirstRevisionID)
	}

	restLocationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+parent, nil, headers)
	if restLocationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workflows get location, got %d body=%s", restLocationResp.StatusCode, string(providerContractBody(t, restLocationResp)))
	}
	restLocationBody := providerContractJSONMap(t, restLocationResp)
	restLocationName, _ := restLocationBody["name"].(string)

	var grpcLocationResp locationpb.Location
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpLocationsGetLocationMethod, &locationpb.GetLocationRequest{
		Name: parent,
	}, &grpcLocationResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for workflows get location, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcLocationResp.GetName() != restLocationName {
		t.Fatalf("expected grpc location %q to match rest %q", grpcLocationResp.GetName(), restLocationName)
	}

	restOpsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+parent+"/operations?pageSize=1", nil, headers)
	if restOpsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workflows list operations, got %d body=%s", restOpsResp.StatusCode, string(providerContractBody(t, restOpsResp)))
	}
	restOpsBody := providerContractJSONMap(t, restOpsResp)
	restOpsItems, ok := restOpsBody["operations"].([]any)
	if !ok || len(restOpsItems) == 0 {
		t.Fatalf("expected operations list in rest payload, got %#v", restOpsBody["operations"])
	}
	restFirstOp, _ := restOpsItems[0].(map[string]any)
	restFirstOpName, _ := restFirstOp["name"].(string)

	var grpcOpsResp longrunningpb.ListOperationsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpLongrunningListOpsMethod, &longrunningpb.ListOperationsRequest{
		Name:     parent,
		PageSize: 1,
	}, &grpcOpsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for workflows list operations, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcOpsResp.GetOperations()) != 1 {
		t.Fatalf("expected one grpc operation, got %d", len(grpcOpsResp.GetOperations()))
	}
	if grpcOpsResp.GetOperations()[0].GetName() != restFirstOpName {
		t.Fatalf("expected grpc operation name %q to match rest %q", grpcOpsResp.GetOperations()[0].GetName(), restFirstOpName)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowsCreateWorkflowMethod, &workflowspb.CreateWorkflowRequest{
		WorkflowId: "workflow-4",
		Workflow: &workflowspb.Workflow{
			Name: parent + "/workflows/workflow-4",
			SourceCode: &workflowspb.Workflow_SourceContents{
				SourceContents: "main:\n  steps:\n  - done:\n      return: \"ok\"",
			},
		},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for workflows create missing parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowsGetWorkflowMethod, &workflowspb.GetWorkflowRequest{
		Name:       workflowName,
		RevisionId: "invalid",
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "revision_id-invalid") {
		t.Fatalf("expected grpc invalid argument for workflows get invalid revision id, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkflowsUpdateWorkflowMethod, &workflowspb.UpdateWorkflowRequest{
		Workflow: &workflowspb.Workflow{
			Name: workflowName,
			SourceCode: &workflowspb.Workflow_SourceContents{
				SourceContents: "main:\n  steps:\n  - done:\n      return: \"updated\"",
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "update_mask-invalid") {
		t.Fatalf("expected grpc invalid argument for workflows update invalid update mask, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
