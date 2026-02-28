package server

type dataPipelineOperation struct {
	Name string
}

// AWS Data Pipeline actions sourced from:
// https://docs.aws.amazon.com/datapipeline/latest/APIReference/API_Operations.html
var dataPipelineOperations = []dataPipelineOperation{
	{Name: "ActivatePipeline"},
	{Name: "AddTags"},
	{Name: "CreatePipeline"},
	{Name: "DeactivatePipeline"},
	{Name: "DeletePipeline"},
	{Name: "DescribeObjects"},
	{Name: "DescribePipelines"},
	{Name: "EvaluateExpression"},
	{Name: "GetPipelineDefinition"},
	{Name: "ListPipelines"},
	{Name: "PollForTask"},
	{Name: "PutPipelineDefinition"},
	{Name: "QueryObjects"},
	{Name: "RemoveTags"},
	{Name: "ReportTaskProgress"},
	{Name: "ReportTaskRunnerHeartbeat"},
	{Name: "SetStatus"},
	{Name: "SetTaskStatus"},
	{Name: "ValidatePipelineDefinition"},
}

var dataPipelineOperationByName = func() map[string]dataPipelineOperation {
	out := make(map[string]dataPipelineOperation, len(dataPipelineOperations))
	for _, op := range dataPipelineOperations {
		out[op.Name] = op
	}
	return out
}()
