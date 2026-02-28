package server

type codeBuildOperation struct {
	Name string
}

// AWS CodeBuild operations sourced from:
// https://docs.aws.amazon.com/codebuild/latest/APIReference/API_Operations.html
var codeBuildOperations = []codeBuildOperation{
	{Name: "BatchDeleteBuilds"},
	{Name: "BatchGetBuildBatches"},
	{Name: "BatchGetBuilds"},
	{Name: "BatchGetCommandExecutions"},
	{Name: "BatchGetFleets"},
	{Name: "BatchGetProjects"},
	{Name: "BatchGetReportGroups"},
	{Name: "BatchGetReports"},
	{Name: "BatchGetSandboxes"},
	{Name: "CreateFleet"},
	{Name: "CreateProject"},
	{Name: "CreateReportGroup"},
	{Name: "CreateWebhook"},
	{Name: "DeleteBuildBatch"},
	{Name: "DeleteFleet"},
	{Name: "DeleteProject"},
	{Name: "DeleteReport"},
	{Name: "DeleteReportGroup"},
	{Name: "DeleteResourcePolicy"},
	{Name: "DeleteSourceCredentials"},
	{Name: "DeleteWebhook"},
	{Name: "DescribeCodeCoverages"},
	{Name: "DescribeTestCases"},
	{Name: "GetReportGroupTrend"},
	{Name: "GetResourcePolicy"},
	{Name: "ImportSourceCredentials"},
	{Name: "InvalidateProjectCache"},
	{Name: "ListBuildBatches"},
	{Name: "ListBuildBatchesForProject"},
	{Name: "ListBuilds"},
	{Name: "ListBuildsForProject"},
	{Name: "ListCommandExecutionsForSandbox"},
	{Name: "ListCuratedEnvironmentImages"},
	{Name: "ListFleets"},
	{Name: "ListProjects"},
	{Name: "ListReportGroups"},
	{Name: "ListReports"},
	{Name: "ListReportsForReportGroup"},
	{Name: "ListSandboxes"},
	{Name: "ListSandboxesForProject"},
	{Name: "ListSharedProjects"},
	{Name: "ListSharedReportGroups"},
	{Name: "ListSourceCredentials"},
	{Name: "PutResourcePolicy"},
	{Name: "RetryBuild"},
	{Name: "RetryBuildBatch"},
	{Name: "StartBuild"},
	{Name: "StartBuildBatch"},
	{Name: "StartCommandExecution"},
	{Name: "StartSandbox"},
	{Name: "StartSandboxConnection"},
	{Name: "StopBuild"},
	{Name: "StopBuildBatch"},
	{Name: "StopSandbox"},
	{Name: "UpdateFleet"},
	{Name: "UpdateProject"},
	{Name: "UpdateProjectVisibility"},
	{Name: "UpdateReportGroup"},
	{Name: "UpdateWebhook"},
}

var codeBuildOperationByName = func() map[string]codeBuildOperation {
	out := make(map[string]codeBuildOperation, len(codeBuildOperations))
	for _, op := range codeBuildOperations {
		out[op.Name] = op
	}
	return out
}()
