package server

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	workflowspb "cloud.google.com/go/workflows/apiv1/workflowspb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpWorkflowsListWorkflowsMethod         = "/google.cloud.workflows.v1.Workflows/ListWorkflows"
	gcpWorkflowsGetWorkflowMethod           = "/google.cloud.workflows.v1.Workflows/GetWorkflow"
	gcpWorkflowsCreateWorkflowMethod        = "/google.cloud.workflows.v1.Workflows/CreateWorkflow"
	gcpWorkflowsDeleteWorkflowMethod        = "/google.cloud.workflows.v1.Workflows/DeleteWorkflow"
	gcpWorkflowsUpdateWorkflowMethod        = "/google.cloud.workflows.v1.Workflows/UpdateWorkflow"
	gcpWorkflowsListWorkflowRevisionsMethod = "/google.cloud.workflows.v1.Workflows/ListWorkflowRevisions"
)

func gcpStage4GRPCWorkflows(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpWorkflowsListWorkflowsMethod:
		return gcpStage4GRPCWorkflowsListWorkflows(grpcReqBody)
	case gcpWorkflowsGetWorkflowMethod:
		return gcpStage4GRPCWorkflowsGetWorkflow(grpcReqBody)
	case gcpWorkflowsCreateWorkflowMethod:
		return gcpStage4GRPCWorkflowsCreateWorkflow(grpcReqBody)
	case gcpWorkflowsDeleteWorkflowMethod:
		return gcpStage4GRPCWorkflowsDeleteWorkflow(grpcReqBody)
	case gcpWorkflowsUpdateWorkflowMethod:
		return gcpStage4GRPCWorkflowsUpdateWorkflow(grpcReqBody)
	case gcpWorkflowsListWorkflowRevisionsMethod:
		return gcpStage4GRPCWorkflowsListWorkflowRevisions(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCWorkflowsListWorkflows(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workflowspb.ListWorkflowsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPWorkflowsParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	filter := strings.TrimSpace(req.GetFilter())
	if !isGCPWorkflowsSupportedFilter(filter) {
		return grpcInvalidArgument("filter-invalid")
	}
	orderBy := strings.TrimSpace(req.GetOrderBy())
	if !isGCPWorkflowsSupportedOrderBy(orderBy) {
		return grpcInvalidArgument("order_by-invalid")
	}

	items := []*workflowspb.Workflow{
		gcpStage4WorkflowsWorkflow(project, location, "workflow-1", "000001-a4d", workflowspb.Workflow_ACTIVE),
		gcpStage4WorkflowsWorkflow(project, location, "workflow-2", "000002-b5e", workflowspb.Workflow_UNAVAILABLE),
	}

	if filter != "" {
		targetState := workflowspb.Workflow_ACTIVE
		if strings.Contains(strings.ToUpper(filter), "UNAVAILABLE") {
			targetState = workflowspb.Workflow_UNAVAILABLE
		}
		filtered := make([]*workflowspb.Workflow, 0, len(items))
		for _, item := range items {
			if item.GetState() == targetState {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if strings.EqualFold(orderBy, "createTime desc") {
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].GetCreateTime().AsTime().After(items[j].GetCreateTime().AsTime())
		})
	}
	if strings.EqualFold(orderBy, "updateTime desc") {
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].GetUpdateTime().AsTime().After(items[j].GetUpdateTime().AsTime())
		})
	}

	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&workflowspb.ListWorkflowsResponse{
		Workflows:     items[start:end],
		NextPageToken: next,
		Unreachable:   []string{},
	})
}

func gcpStage4GRPCWorkflowsGetWorkflow(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workflowspb.GetWorkflowRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, workflowID, ok := parseGCPWorkflowsWorkflowName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	revisionID := strings.TrimSpace(req.GetRevisionId())
	if revisionID != "" && !gcpWorkflowsRevisionRegex.MatchString(revisionID) {
		return grpcInvalidArgument("revision_id-invalid")
	}
	if revisionID == "" {
		revisionID = "000001-a4d"
	}
	return grpcProtoSuccess(gcpStage4WorkflowsWorkflow(project, location, workflowID, revisionID, workflowspb.Workflow_ACTIVE))
}

func gcpStage4GRPCWorkflowsCreateWorkflow(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workflowspb.CreateWorkflowRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPWorkflowsParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	workflowID := strings.TrimSpace(req.GetWorkflowId())
	if workflowID == "" {
		return grpcInvalidArgument("workflow_id-required")
	}
	if !isGCPWorkflowsID(workflowID) {
		return grpcInvalidArgument("workflow_id-invalid")
	}
	if strings.Contains(strings.ToLower(workflowID), "existing") {
		return grpcAlreadyExists("workflow-already-exists")
	}
	workflow := req.GetWorkflow()
	if workflow == nil {
		return grpcInvalidArgument("workflow-required")
	}
	if ok, reason := gcpStage4ValidateWorkflowsWorkflow(workflow, false); !ok {
		return grpcInvalidArgument(reason)
	}

	expectedName := gcpWorkflowsWorkflowName(project, location, workflowID)
	if name := strings.TrimSpace(workflow.GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("workflow_name-mismatch")
	}

	return grpcProtoSuccess(gcpStage4WorkflowsOperation(project, location, "createWorkflow."+workflowID, expectedName, "create", false))
}

func gcpStage4GRPCWorkflowsDeleteWorkflow(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workflowspb.DeleteWorkflowRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, workflowID, ok := parseGCPWorkflowsWorkflowName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(workflowID), "missing") {
		return grpcNotFound("workflow-not-found")
	}
	return grpcProtoSuccess(gcpStage4WorkflowsOperation(project, location, "deleteWorkflow."+workflowID, gcpWorkflowsWorkflowName(project, location, workflowID), "delete", false))
}

func gcpStage4GRPCWorkflowsUpdateWorkflow(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workflowspb.UpdateWorkflowRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	workflow := req.GetWorkflow()
	if workflow == nil {
		return grpcInvalidArgument("workflow-required")
	}
	if ok, reason := gcpStage4ValidateWorkflowsWorkflow(workflow, true); !ok {
		return grpcInvalidArgument(reason)
	}
	project, location, workflowID, ok := parseGCPWorkflowsWorkflowName(strings.TrimSpace(workflow.GetName()))
	if !ok {
		return grpcInvalidArgument("workflow_name-invalid")
	}
	if strings.Contains(strings.ToLower(workflowID), "missing") {
		return grpcNotFound("workflow-not-found")
	}
	if updateMask := req.GetUpdateMask(); updateMask != nil {
		if len(updateMask.GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-invalid")
		}
		for _, path := range updateMask.GetPaths() {
			if strings.TrimSpace(path) == "" {
				return grpcInvalidArgument("update_mask-invalid")
			}
		}
	}
	return grpcProtoSuccess(gcpStage4WorkflowsOperation(project, location, "updateWorkflow."+workflowID, gcpWorkflowsWorkflowName(project, location, workflowID), "update", false))
}

func gcpStage4GRPCWorkflowsListWorkflowRevisions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workflowspb.ListWorkflowRevisionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, workflowID, ok := parseGCPWorkflowsWorkflowName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 100 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}

	items := []*workflowspb.Workflow{
		gcpStage4WorkflowsWorkflow(project, location, workflowID, "000003-c6f", workflowspb.Workflow_ACTIVE),
		gcpStage4WorkflowsWorkflow(project, location, workflowID, "000002-b5e", workflowspb.Workflow_ACTIVE),
		gcpStage4WorkflowsWorkflow(project, location, workflowID, "000001-a4d", workflowspb.Workflow_ACTIVE),
	}

	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&workflowspb.ListWorkflowRevisionsResponse{
		Workflows:     items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4ValidateWorkflowsWorkflow(workflow *workflowspb.Workflow, requireName bool) (bool, string) {
	if workflow == nil {
		return false, "workflow-required"
	}
	name := strings.TrimSpace(workflow.GetName())
	if requireName && name == "" {
		return false, "workflow_name-required"
	}
	if name != "" {
		if _, _, _, ok := parseGCPWorkflowsWorkflowName(name); !ok {
			return false, "workflow_name-invalid"
		}
	}
	if len(strings.TrimSpace(workflow.GetDescription())) > 1000 {
		return false, "workflow_description-too-long"
	}
	if strings.TrimSpace(workflow.GetSourceContents()) == "" {
		return false, "workflow_source_contents-required"
	}
	if len(workflow.GetLabels()) > 64 {
		return false, "workflow_labels-too-many"
	}
	for key, value := range workflow.GetLabels() {
		key = strings.TrimSpace(key)
		if key == "" || len(key) > 63 {
			return false, "workflow_label_key-invalid"
		}
		if len(strings.TrimSpace(value)) > 63 {
			return false, "workflow_label_value-invalid"
		}
	}
	switch workflow.GetCallLogLevel() {
	case workflowspb.Workflow_CALL_LOG_LEVEL_UNSPECIFIED,
		workflowspb.Workflow_LOG_ALL_CALLS,
		workflowspb.Workflow_LOG_ERRORS_ONLY,
		workflowspb.Workflow_LOG_NONE:
	default:
		return false, "workflow_call_log_level-invalid"
	}
	switch workflow.GetExecutionHistoryLevel() {
	case workflowspb.ExecutionHistoryLevel_EXECUTION_HISTORY_LEVEL_UNSPECIFIED,
		workflowspb.ExecutionHistoryLevel_EXECUTION_HISTORY_BASIC,
		workflowspb.ExecutionHistoryLevel_EXECUTION_HISTORY_DETAILED:
	default:
		return false, "workflow_execution_history_level-invalid"
	}
	return true, ""
}

func gcpStage4WorkflowsWorkflow(project, location, workflowID, revisionID string, state workflowspb.Workflow_State) *workflowspb.Workflow {
	if revisionID == "" {
		revisionID = "000001-a4d"
	}
	createTime := gcpStage4ReferenceTime
	updateTime := createTime.Add(10 * time.Minute)
	return &workflowspb.Workflow{
		Name:               gcpWorkflowsWorkflowName(project, location, workflowID),
		Description:        "Stackyard staged workflow " + workflowID,
		State:              state,
		RevisionId:         revisionID,
		CreateTime:         timestamppb.New(createTime),
		UpdateTime:         timestamppb.New(updateTime),
		RevisionCreateTime: timestamppb.New(updateTime),
		Labels: map[string]string{
			"env":      "staged",
			"service":  "workflows",
			"provider": providerGCP,
		},
		ServiceAccount: fmt.Sprintf("workflow-sa@%s.iam.gserviceaccount.com", project),
		SourceCode: &workflowspb.Workflow_SourceContents{
			SourceContents: "main:\n  params: [input]\n  steps:\n  - return_output:\n      return: ${input}",
		},
		CallLogLevel: workflowspb.Workflow_LOG_ERRORS_ONLY,
		UserEnvVars: map[string]string{
			"STACKYARD": "true",
		},
		ExecutionHistoryLevel: workflowspb.ExecutionHistoryLevel_EXECUTION_HISTORY_BASIC,
	}
}

func gcpStage4WorkflowsOperation(project, location, operationID, target, verb string, done bool) *longrunningpb.Operation {
	metadataAny, err := anypb.New(&workflowspb.OperationMetadata{
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		EndTime:    timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Minute)),
		Target:     target,
		Verb:       verb,
		ApiVersion: "v1",
	})
	if err != nil {
		metadataAny = nil
	}

	op := &longrunningpb.Operation{
		Name:     fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Metadata: metadataAny,
		Done:     done,
	}

	if !done {
		return op
	}

	switch verb {
	case "delete":
		responseAny, err := anypb.New(&emptypb.Empty{})
		if err == nil {
			op.Result = &longrunningpb.Operation_Response{Response: responseAny}
		}
	default:
		responseAny, err := anypb.New(&workflowspb.Workflow{Name: target})
		if err == nil {
			op.Result = &longrunningpb.Operation_Response{Response: responseAny}
		}
	}
	return op
}
