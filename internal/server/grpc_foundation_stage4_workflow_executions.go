package server

import (
	"sort"
	"strconv"
	"strings"
	"time"

	executionspb "cloud.google.com/go/workflows/executions/apiv1/executionspb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpWorkflowExecutionsListExecutionsMethod  = "/google.cloud.workflows.executions.v1.Executions/ListExecutions"
	gcpWorkflowExecutionsCreateExecutionMethod = "/google.cloud.workflows.executions.v1.Executions/CreateExecution"
	gcpWorkflowExecutionsGetExecutionMethod    = "/google.cloud.workflows.executions.v1.Executions/GetExecution"
	gcpWorkflowExecutionsCancelExecutionMethod = "/google.cloud.workflows.executions.v1.Executions/CancelExecution"
)

func gcpStage4GRPCWorkflowExecutions(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpWorkflowExecutionsListExecutionsMethod:
		return gcpStage4GRPCWorkflowExecutionsListExecutions(grpcReqBody)
	case gcpWorkflowExecutionsCreateExecutionMethod:
		return gcpStage4GRPCWorkflowExecutionsCreateExecution(grpcReqBody)
	case gcpWorkflowExecutionsGetExecutionMethod:
		return gcpStage4GRPCWorkflowExecutionsGetExecution(grpcReqBody)
	case gcpWorkflowExecutionsCancelExecutionMethod:
		return gcpStage4GRPCWorkflowExecutionsCancelExecution(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCWorkflowExecutionsListExecutions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &executionspb.ListExecutionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}

	project, location, workflowID, ok := parseGCPWorkflowExecutionsParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}

	fullView, maxPageSize, ok := gcpStage4WorkflowExecutionsListView(req.GetView())
	if !ok {
		return grpcInvalidArgument("view-invalid")
	}
	if req.GetPageSize() < 0 || int(req.GetPageSize()) > maxPageSize {
		return grpcInvalidArgument("page_size-invalid")
	}

	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}

	filterState, ok := parseGCPWorkflowExecutionsStateFilter(strings.TrimSpace(req.GetFilter()))
	if !ok {
		return grpcInvalidArgument("filter-invalid")
	}
	orderBy := strings.TrimSpace(req.GetOrderBy())
	if !isGCPWorkflowExecutionsSupportedOrderBy(orderBy) {
		return grpcInvalidArgument("order_by-invalid")
	}

	items := []*executionspb.Execution{
		gcpStage4WorkflowExecutionsExecution(project, location, workflowID, "execution-1", executionspb.Execution_ACTIVE, fullView),
		gcpStage4WorkflowExecutionsExecution(project, location, workflowID, "execution-2", executionspb.Execution_SUCCEEDED, fullView),
		gcpStage4WorkflowExecutionsExecution(project, location, workflowID, "execution-3", executionspb.Execution_FAILED, fullView),
	}
	if filterState != "" {
		targetState := gcpStage4WorkflowExecutionsStateFromString(filterState)
		filtered := make([]*executionspb.Execution, 0, len(items))
		for _, item := range items {
			if item.GetState() == targetState {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	gcpStage4ApplyWorkflowExecutionsOrdering(items, orderBy)

	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}

	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}

	return grpcProtoSuccess(&executionspb.ListExecutionsResponse{
		Executions:    items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCWorkflowExecutionsCreateExecution(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &executionspb.CreateExecutionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}

	project, location, workflowID, ok := parseGCPWorkflowExecutionsParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	execution := req.GetExecution()
	if execution == nil {
		return grpcInvalidArgument("execution-required")
	}
	if ok, reason := gcpStage4ValidateWorkflowExecutionCreateInput(execution); !ok {
		return grpcInvalidArgument(reason)
	}

	executionID := "execution-created"
	if name := strings.TrimSpace(execution.GetName()); name != "" {
		nameProject, nameLocation, nameWorkflowID, nameExecutionID, parsed := parseGCPWorkflowExecutionsName(name)
		if !parsed {
			return grpcInvalidArgument("execution_name-invalid")
		}
		if nameProject != project || nameLocation != location || nameWorkflowID != workflowID {
			return grpcInvalidArgument("execution_name-parent-mismatch")
		}
		executionID = nameExecutionID
	}
	if strings.Contains(strings.ToLower(executionID), "existing") {
		return grpcAlreadyExists("execution-already-exists")
	}

	out := gcpStage4WorkflowExecutionsExecution(project, location, workflowID, executionID, executionspb.Execution_ACTIVE, true)
	if argument := strings.TrimSpace(execution.GetArgument()); argument != "" {
		out.Argument = argument
	}
	if execution.GetCallLogLevel() != executionspb.Execution_CALL_LOG_LEVEL_UNSPECIFIED {
		out.CallLogLevel = execution.GetCallLogLevel()
	}
	if len(execution.GetLabels()) > 0 {
		out.Labels = map[string]string{}
		for key, value := range execution.GetLabels() {
			out.Labels[key] = value
		}
	}
	return grpcProtoSuccess(out)
}

func gcpStage4GRPCWorkflowExecutionsGetExecution(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &executionspb.GetExecutionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}

	project, location, workflowID, executionID, ok := parseGCPWorkflowExecutionsName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(executionID), "missing") {
		return grpcNotFound("execution-not-found")
	}
	fullView, ok := gcpStage4WorkflowExecutionsGetView(req.GetView())
	if !ok {
		return grpcInvalidArgument("view-invalid")
	}

	state := gcpStage4WorkflowExecutionsStateForID(executionID)
	return grpcProtoSuccess(gcpStage4WorkflowExecutionsExecution(project, location, workflowID, executionID, state, fullView))
}

func gcpStage4GRPCWorkflowExecutionsCancelExecution(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &executionspb.CancelExecutionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}

	project, location, workflowID, executionID, ok := parseGCPWorkflowExecutionsName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(executionID), "missing") {
		return grpcNotFound("execution-not-found")
	}
	state := gcpStage4WorkflowExecutionsStateForID(executionID)
	switch state {
	case executionspb.Execution_SUCCEEDED, executionspb.Execution_FAILED, executionspb.Execution_CANCELLED:
		return grpcFailedPrecondition("execution-not-cancellable")
	}

	return grpcProtoSuccess(gcpStage4WorkflowExecutionsExecution(project, location, workflowID, executionID, executionspb.Execution_CANCELLED, true))
}

func gcpStage4ValidateWorkflowExecutionCreateInput(execution *executionspb.Execution) (bool, string) {
	if execution == nil {
		return false, "execution-required"
	}
	if len(strings.TrimSpace(execution.GetArgument())) > 32768 {
		return false, "execution_argument-too-long"
	}
	switch execution.GetCallLogLevel() {
	case executionspb.Execution_CALL_LOG_LEVEL_UNSPECIFIED,
		executionspb.Execution_LOG_ALL_CALLS,
		executionspb.Execution_LOG_ERRORS_ONLY,
		executionspb.Execution_LOG_NONE:
	default:
		return false, "execution_call_log_level-invalid"
	}
	if len(execution.GetLabels()) > 64 {
		return false, "execution_labels-too-many"
	}
	for key, value := range execution.GetLabels() {
		if !gcpWorkflowExecutionsLabelKeyRegex.MatchString(strings.TrimSpace(key)) {
			return false, "execution_labels-key-invalid"
		}
		if !gcpWorkflowExecutionsLabelValueRegex.MatchString(strings.TrimSpace(value)) {
			return false, "execution_labels-value-invalid"
		}
	}
	if execution.GetResult() != "" {
		return false, "execution_result-output-only"
	}
	if execution.GetError() != nil {
		return false, "execution_error-output-only"
	}
	if execution.GetStatus() != nil {
		return false, "execution_status-output-only"
	}
	if execution.GetStateError() != nil {
		return false, "execution_state_error-output-only"
	}
	if execution.GetWorkflowRevisionId() != "" {
		return false, "execution_workflow_revision_id-output-only"
	}
	if execution.GetState() != executionspb.Execution_STATE_UNSPECIFIED && execution.GetState() != executionspb.Execution_ACTIVE {
		return false, "execution_state-output-only"
	}
	return true, ""
}

func gcpStage4WorkflowExecutionsListView(view executionspb.ExecutionView) (full bool, maxPageSize int, ok bool) {
	switch view {
	case executionspb.ExecutionView_EXECUTION_VIEW_UNSPECIFIED, executionspb.ExecutionView_BASIC:
		return false, 1000, true
	case executionspb.ExecutionView_FULL:
		return true, 100, true
	default:
		return false, 0, false
	}
}

func gcpStage4WorkflowExecutionsGetView(view executionspb.ExecutionView) (full bool, ok bool) {
	switch view {
	case executionspb.ExecutionView_EXECUTION_VIEW_UNSPECIFIED, executionspb.ExecutionView_FULL:
		return true, true
	case executionspb.ExecutionView_BASIC:
		return false, true
	default:
		return false, false
	}
}

func gcpStage4WorkflowExecutionsStateFromString(state string) executionspb.Execution_State {
	switch strings.TrimSpace(strings.ToUpper(state)) {
	case "SUCCEEDED":
		return executionspb.Execution_SUCCEEDED
	case "FAILED":
		return executionspb.Execution_FAILED
	case "CANCELLED":
		return executionspb.Execution_CANCELLED
	case "QUEUED":
		return executionspb.Execution_QUEUED
	case "UNAVAILABLE":
		return executionspb.Execution_UNAVAILABLE
	default:
		return executionspb.Execution_ACTIVE
	}
}

func gcpStage4WorkflowExecutionsStateForID(executionID string) executionspb.Execution_State {
	switch gcpWorkflowExecutionsStateForExecutionID(executionID) {
	case "SUCCEEDED":
		return executionspb.Execution_SUCCEEDED
	case "FAILED":
		return executionspb.Execution_FAILED
	case "CANCELLED":
		return executionspb.Execution_CANCELLED
	case "QUEUED":
		return executionspb.Execution_QUEUED
	case "UNAVAILABLE":
		return executionspb.Execution_UNAVAILABLE
	default:
		return executionspb.Execution_ACTIVE
	}
}

func gcpStage4ApplyWorkflowExecutionsOrdering(items []*executionspb.Execution, orderBy string) {
	normalized := strings.TrimSpace(strings.ToLower(orderBy))
	if normalized == "" {
		normalized = "starttime desc"
	}
	parts := strings.Fields(normalized)
	field := parts[0]
	desc := len(parts) > 1 && parts[1] == "desc"

	sort.SliceStable(items, func(i, j int) bool {
		cmp := gcpStage4CompareWorkflowExecution(items[i], items[j], field)
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func gcpStage4CompareWorkflowExecution(left, right *executionspb.Execution, field string) int {
	switch field {
	case "state":
		return strings.Compare(left.GetState().String(), right.GetState().String())
	case "endtime":
		return gcpStage4CompareTime(left.GetEndTime().AsTime(), right.GetEndTime().AsTime())
	default:
		return gcpStage4CompareTime(left.GetStartTime().AsTime(), right.GetStartTime().AsTime())
	}
}

func gcpStage4CompareTime(left, right time.Time) int {
	switch {
	case left.Before(right):
		return -1
	case left.After(right):
		return 1
	default:
		return 0
	}
}

func gcpStage4WorkflowExecutionsExecution(project, location, workflowID, executionID string, state executionspb.Execution_State, full bool) *executionspb.Execution {
	index := 1
	switch executionID {
	case "execution-1":
		index = 1
	case "execution-2":
		index = 2
	case "execution-3":
		index = 3
	default:
		index = 4
	}

	start := gcpWorkflowExecutionsReferenceTime.Add(time.Duration(index) * time.Minute)
	end := start.Add(45 * time.Second)
	out := &executionspb.Execution{
		Name:               gcpWorkflowExecutionsExecutionName(project, location, workflowID, executionID),
		StartTime:          timestamppb.New(start),
		EndTime:            timestamppb.New(end),
		Duration:           durationpb.New(45 * time.Second),
		State:              state,
		WorkflowRevisionId: "000001-a4d",
		CallLogLevel:       executionspb.Execution_LOG_ERRORS_ONLY,
	}

	if !full {
		return out
	}

	out.Argument = `{"input":"` + executionID + `"}`
	out.Result = ""
	out.Error = &executionspb.Execution_Error{
		Payload: `{"message":""}`,
		Context: "",
		StackTrace: &executionspb.Execution_StackTrace{
			Elements: []*executionspb.Execution_StackTraceElement{},
		},
	}
	out.Status = &executionspb.Execution_Status{
		CurrentSteps: []*executionspb.Execution_Status_Step{
			{
				Routine: "main",
				Step:    "run-step",
			},
		},
	}
	out.Labels = map[string]string{
		"env":      "staged",
		"service":  "workflow_executions",
		"provider": providerGCP,
	}

	switch state {
	case executionspb.Execution_SUCCEEDED:
		out.Result = `{"status":"ok"}`
	case executionspb.Execution_FAILED:
		out.Error = &executionspb.Execution_Error{
			Payload: `{"message":"workflow failed"}`,
			Context: "main.run-step",
			StackTrace: &executionspb.Execution_StackTrace{
				Elements: []*executionspb.Execution_StackTraceElement{
					{
						Routine: "main",
						Step:    "run-step",
					},
				},
			},
		}
	case executionspb.Execution_CANCELLED:
		out.Error = &executionspb.Execution_Error{
			Payload: `{"message":"execution cancelled"}`,
			Context: "cancelled by user",
			StackTrace: &executionspb.Execution_StackTrace{
				Elements: []*executionspb.Execution_StackTraceElement{},
			},
		}
	case executionspb.Execution_UNAVAILABLE:
		out.StateError = &executionspb.Execution_StateError{
			Details: "KMS key permission revoked",
			Type:    executionspb.Execution_StateError_KMS_ERROR,
		}
	}

	return out
}
