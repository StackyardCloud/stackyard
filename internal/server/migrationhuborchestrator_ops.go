package server

type migrationHubOrchestratorOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Migration Hub Orchestrator operations sourced from:
// https://docs.aws.amazon.com/migrationhub-orchestrator/latest/APIReference/API_Operations.html
var migrationHubOrchestratorOperations = []migrationHubOrchestratorOperation{
	{Name: "CreateTemplate", Method: "POST", URI: "/template"},
	{Name: "CreateWorkflow", Method: "POST", URI: "/migrationworkflow/"},
	{Name: "CreateWorkflowStep", Method: "POST", URI: "/workflowstep"},
	{Name: "CreateWorkflowStepGroup", Method: "POST", URI: "/workflowstepgroups"},
	{Name: "DeleteTemplate", Method: "DELETE", URI: "/template/{id}"},
	{Name: "DeleteWorkflow", Method: "DELETE", URI: "/migrationworkflow/{id}"},
	{Name: "DeleteWorkflowStep", Method: "DELETE", URI: "/workflowstep/{id}"},
	{Name: "DeleteWorkflowStepGroup", Method: "DELETE", URI: "/workflowstepgroup/{id}"},
	{Name: "GetTemplate", Method: "GET", URI: "/migrationworkflowtemplate/{id}"},
	{Name: "GetTemplateStep", Method: "GET", URI: "/templatestep/{id}"},
	{Name: "GetTemplateStepGroup", Method: "GET", URI: "/templates/{templateId}/stepgroups/{id}"},
	{Name: "GetWorkflow", Method: "GET", URI: "/migrationworkflow/{id}"},
	{Name: "GetWorkflowStep", Method: "GET", URI: "/workflowstep/{id}"},
	{Name: "GetWorkflowStepGroup", Method: "GET", URI: "/workflowstepgroup/{id}"},
	{Name: "ListPlugins", Method: "GET", URI: "/plugins"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListTemplateStepGroups", Method: "GET", URI: "/templatestepgroups/{templateId}"},
	{Name: "ListTemplateSteps", Method: "GET", URI: "/templatesteps"},
	{Name: "ListTemplates", Method: "GET", URI: "/migrationworkflowtemplates"},
	{Name: "ListWorkflowStepGroups", Method: "GET", URI: "/workflowstepgroups"},
	{Name: "ListWorkflowSteps", Method: "GET", URI: "/workflow/{workflowId}/workflowstepgroups/{stepGroupId}/workflowsteps"},
	{Name: "ListWorkflows", Method: "GET", URI: "/migrationworkflows"},
	{Name: "RetryWorkflowStep", Method: "POST", URI: "/retryworkflowstep/{id}"},
	{Name: "StartWorkflow", Method: "POST", URI: "/migrationworkflow/{id}/start"},
	{Name: "StopWorkflow", Method: "POST", URI: "/migrationworkflow/{id}/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateTemplate", Method: "POST", URI: "/template/{id}"},
	{Name: "UpdateWorkflow", Method: "POST", URI: "/migrationworkflow/{id}"},
	{Name: "UpdateWorkflowStep", Method: "POST", URI: "/workflowstep/{id}"},
	{Name: "UpdateWorkflowStepGroup", Method: "POST", URI: "/workflowstepgroup/{id}"},
}

var migrationHubOrchestratorOperationByName = func() map[string]migrationHubOrchestratorOperation {
	out := make(map[string]migrationHubOrchestratorOperation, len(migrationHubOrchestratorOperations))
	for _, op := range migrationHubOrchestratorOperations {
		out[op.Name] = op
	}
	return out
}()
