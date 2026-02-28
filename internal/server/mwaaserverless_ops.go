package server

type mwaaServerlessOperation struct {
	Name string
}

// Amazon Managed Workflows for Apache Airflow Serverless actions sourced from:
// https://docs.aws.amazon.com/mwaa-serverless/latest/APIReference/API_Operations.html
var mwaaServerlessOperations = []mwaaServerlessOperation{
	{Name: "CreateWorkflow"},
	{Name: "DeleteWorkflow"},
	{Name: "GetTaskInstance"},
	{Name: "GetWorkflow"},
	{Name: "GetWorkflowRun"},
	{Name: "ListTagsForResource"},
	{Name: "ListTaskInstances"},
	{Name: "ListWorkflowRuns"},
	{Name: "ListWorkflowVersions"},
	{Name: "ListWorkflows"},
	{Name: "StartWorkflowRun"},
	{Name: "StopWorkflowRun"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateWorkflow"},
}

var mwaaServerlessOperationByName = func() map[string]mwaaServerlessOperation {
	out := make(map[string]mwaaServerlessOperation, len(mwaaServerlessOperations))
	for _, op := range mwaaServerlessOperations {
		out[op.Name] = op
	}
	return out
}()
