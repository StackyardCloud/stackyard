package server

type codePipelineOperation struct {
	Name string
}

// AWS CodePipeline operations sourced from:
// https://docs.aws.amazon.com/codepipeline/latest/APIReference/API_Operations.html
var codePipelineOperations = []codePipelineOperation{
	{Name: "AcknowledgeJob"},
	{Name: "AcknowledgeThirdPartyJob"},
	{Name: "CreateCustomActionType"},
	{Name: "CreatePipeline"},
	{Name: "DeleteCustomActionType"},
	{Name: "DeletePipeline"},
	{Name: "DeleteWebhook"},
	{Name: "DeregisterWebhookWithThirdParty"},
	{Name: "DisableStageTransition"},
	{Name: "EnableStageTransition"},
	{Name: "GetActionType"},
	{Name: "GetJobDetails"},
	{Name: "GetPipeline"},
	{Name: "GetPipelineExecution"},
	{Name: "GetPipelineState"},
	{Name: "GetThirdPartyJobDetails"},
	{Name: "ListActionExecutions"},
	{Name: "ListActionTypes"},
	{Name: "ListDeployActionExecutionTargets"},
	{Name: "ListPipelineExecutions"},
	{Name: "ListPipelines"},
	{Name: "ListRuleExecutions"},
	{Name: "ListRuleTypes"},
	{Name: "ListTagsForResource"},
	{Name: "ListWebhooks"},
	{Name: "OverrideStageCondition"},
	{Name: "PollForJobs"},
	{Name: "PollForThirdPartyJobs"},
	{Name: "PutActionRevision"},
	{Name: "PutApprovalResult"},
	{Name: "PutJobFailureResult"},
	{Name: "PutJobSuccessResult"},
	{Name: "PutThirdPartyJobFailureResult"},
	{Name: "PutThirdPartyJobSuccessResult"},
	{Name: "PutWebhook"},
	{Name: "RegisterWebhookWithThirdParty"},
	{Name: "RetryStageExecution"},
	{Name: "RollbackStage"},
	{Name: "StartPipelineExecution"},
	{Name: "StopPipelineExecution"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateActionType"},
	{Name: "UpdatePipeline"},
}

var codePipelineOperationByName = func() map[string]codePipelineOperation {
	out := make(map[string]codePipelineOperation, len(codePipelineOperations))
	for _, op := range codePipelineOperations {
		out[op.Name] = op
	}
	return out
}()
