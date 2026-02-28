package server

type m2Operation struct {
	Name   string
	Method string
	URI    string
}

// AWS Mainframe Modernization operations sourced from:
// https://docs.aws.amazon.com/m2/latest/APIReference/API_Operations.html
var m2Operations = []m2Operation{
	{Name: "CancelBatchJobExecution", Method: "POST", URI: "/applications/{applicationId}/batch-job-executions/{executionId}/cancel"},
	{Name: "CreateApplication", Method: "POST", URI: "/applications"},
	{Name: "CreateDataSetExportTask", Method: "POST", URI: "/applications/{applicationId}/dataset-export-task"},
	{Name: "CreateDataSetImportTask", Method: "POST", URI: "/applications/{applicationId}/dataset-import-task"},
	{Name: "CreateDeployment", Method: "POST", URI: "/applications/{applicationId}/deployments"},
	{Name: "CreateEnvironment", Method: "POST", URI: "/environments"},
	{Name: "DeleteApplication", Method: "DELETE", URI: "/applications/{applicationId}"},
	{Name: "DeleteApplicationFromEnvironment", Method: "DELETE", URI: "/applications/{applicationId}/environment/{environmentId}"},
	{Name: "DeleteEnvironment", Method: "DELETE", URI: "/environments/{environmentId}"},
	{Name: "GetApplication", Method: "GET", URI: "/applications/{applicationId}"},
	{Name: "GetApplicationVersion", Method: "GET", URI: "/applications/{applicationId}/versions/{applicationVersion}"},
	{Name: "GetBatchJobExecution", Method: "GET", URI: "/applications/{applicationId}/batch-job-executions/{executionId}"},
	{Name: "GetDataSetDetails", Method: "GET", URI: "/applications/{applicationId}/datasets/{dataSetName}"},
	{Name: "GetDataSetExportTask", Method: "GET", URI: "/applications/{applicationId}/dataset-export-tasks/{taskId}"},
	{Name: "GetDataSetImportTask", Method: "GET", URI: "/applications/{applicationId}/dataset-import-tasks/{taskId}"},
	{Name: "GetDeployment", Method: "GET", URI: "/applications/{applicationId}/deployments/{deploymentId}"},
	{Name: "GetEnvironment", Method: "GET", URI: "/environments/{environmentId}"},
	{Name: "GetSignedBluinsightsUrl", Method: "GET", URI: "/signed-bi-url"},
	{Name: "ListApplicationVersions", Method: "GET", URI: "/applications/{applicationId}/versions"},
	{Name: "ListApplications", Method: "GET", URI: "/applications"},
	{Name: "ListBatchJobDefinitions", Method: "GET", URI: "/applications/{applicationId}/batch-job-definitions"},
	{Name: "ListBatchJobExecutions", Method: "GET", URI: "/applications/{applicationId}/batch-job-executions"},
	{Name: "ListBatchJobRestartPoints", Method: "GET", URI: "/applications/{applicationId}/batch-job-executions/{executionId}/steps"},
	{Name: "ListDataSetExportHistory", Method: "GET", URI: "/applications/{applicationId}/dataset-export-tasks"},
	{Name: "ListDataSetImportHistory", Method: "GET", URI: "/applications/{applicationId}/dataset-import-tasks"},
	{Name: "ListDataSets", Method: "GET", URI: "/applications/{applicationId}/datasets"},
	{Name: "ListDeployments", Method: "GET", URI: "/applications/{applicationId}/deployments"},
	{Name: "ListEngineVersions", Method: "GET", URI: "/engine-versions"},
	{Name: "ListEnvironments", Method: "GET", URI: "/environments"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "StartApplication", Method: "POST", URI: "/applications/{applicationId}/start"},
	{Name: "StartBatchJob", Method: "POST", URI: "/applications/{applicationId}/batch-job"},
	{Name: "StopApplication", Method: "POST", URI: "/applications/{applicationId}/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateApplication", Method: "PATCH", URI: "/applications/{applicationId}"},
	{Name: "UpdateEnvironment", Method: "PATCH", URI: "/environments/{environmentId}"},
}

var m2OperationByName = func() map[string]m2Operation {
	out := make(map[string]m2Operation, len(m2Operations))
	for _, op := range m2Operations {
		out[op.Name] = op
	}
	return out
}()
