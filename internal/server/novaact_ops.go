package server

type novaActOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Nova Act operations sourced from:
// https://docs.aws.amazon.com/nova-act/latest/APIReference/API_Operations.html
var novaActOperations = []novaActOperation{
	{Name: "CreateAct", Method: "PUT", URI: "/workflow-definitions/{workflowDefinitionName}/workflow-runs/{workflowRunId}/sessions/{sessionId}/acts"},
	{Name: "CreateSession", Method: "PUT", URI: "/workflow-definitions/{workflowDefinitionName}/workflow-runs/{workflowRunId}/sessions"},
	{Name: "CreateWorkflowDefinition", Method: "PUT", URI: "/workflow-definitions"},
	{Name: "CreateWorkflowRun", Method: "PUT", URI: "/workflow-definitions/{workflowDefinitionName}/workflow-runs"},
	{Name: "DeleteWorkflowDefinition", Method: "DELETE", URI: "/workflow-definitions/{workflowDefinitionName}"},
	{Name: "DeleteWorkflowRun", Method: "DELETE", URI: "/workflow-definitions/{workflowDefinitionName}/workflow-runs/{workflowRunId}"},
	{Name: "GetWorkflowDefinition", Method: "GET", URI: "/workflow-definitions/{workflowDefinitionName}"},
	{Name: "GetWorkflowRun", Method: "GET", URI: "/workflow-definitions/{workflowDefinitionName}/workflow-runs/{workflowRunId}"},
	{Name: "InvokeActStep", Method: "PUT", URI: "/workflow-definitions/{workflowDefinitionName}/workflow-runs/{workflowRunId}/sessions/{sessionId}/acts/{actId}/invoke-step/"},
	{Name: "ListActs", Method: "POST", URI: "/workflow-definitions/{workflowDefinitionName}/acts?maxResults={maxResults}&nextToken={nextToken}&sessionId={sessionId}&workflowRunId={workflowRunId}"},
	{Name: "ListModels", Method: "POST", URI: "/models?clientCompatibilityVersion={clientCompatibilityVersion}"},
	{Name: "ListSessions", Method: "POST", URI: "/workflow-definitions/{workflowDefinitionName}/workflow-runs/{workflowRunId}?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListWorkflowDefinitions", Method: "POST", URI: "/workflow-definitions?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListWorkflowRuns", Method: "POST", URI: "/workflow-definitions/{workflowDefinitionName}/workflow-runs?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "UpdateAct", Method: "PUT", URI: "/workflow-definitions/{workflowDefinitionName}/workflow-runs/{workflowRunId}/sessions/{sessionId}/acts/{actId}"},
	{Name: "UpdateWorkflowRun", Method: "PUT", URI: "/workflow-definitions/{workflowDefinitionName}/workflow-runs/{workflowRunId}"},
}

var novaActOperationByName = func() map[string]novaActOperation {
	out := make(map[string]novaActOperation, len(novaActOperations))
	for _, op := range novaActOperations {
		out[op.Name] = op
	}
	return out
}()
