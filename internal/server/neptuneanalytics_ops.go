package server

type neptuneAnalyticsOperation struct {
	Name string
}

// Neptune Analytics API operations sourced from:
// https://docs.aws.amazon.com/neptune-analytics/latest/apiref/API_Operations.html
var neptuneAnalyticsOperations = []neptuneAnalyticsOperation{
	{Name: "CancelExportTask"},
	{Name: "CancelImportTask"},
	{Name: "CancelQuery"},
	{Name: "CreateGraph"},
	{Name: "CreateGraphSnapshot"},
	{Name: "CreateGraphUsingImportTask"},
	{Name: "CreatePrivateGraphEndpoint"},
	{Name: "DeleteGraph"},
	{Name: "DeleteGraphSnapshot"},
	{Name: "DeletePrivateGraphEndpoint"},
	{Name: "ExecuteQuery"},
	{Name: "GetExportTask"},
	{Name: "GetGraph"},
	{Name: "GetGraphSnapshot"},
	{Name: "GetGraphSummary"},
	{Name: "GetImportTask"},
	{Name: "GetPrivateGraphEndpoint"},
	{Name: "GetQuery"},
	{Name: "ListExportTasks"},
	{Name: "ListGraphs"},
	{Name: "ListGraphSnapshots"},
	{Name: "ListImportTasks"},
	{Name: "ListPrivateGraphEndpoints"},
	{Name: "ListQueries"},
	{Name: "ListTagsForResource"},
	{Name: "ResetGraph"},
	{Name: "RestoreGraphFromSnapshot"},
	{Name: "StartExportTask"},
	{Name: "StartGraph"},
	{Name: "StartImportTask"},
	{Name: "StopGraph"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateGraph"},
}

var neptuneAnalyticsOperationByName = func() map[string]neptuneAnalyticsOperation {
	out := make(map[string]neptuneAnalyticsOperation, len(neptuneAnalyticsOperations))
	for _, op := range neptuneAnalyticsOperations {
		out[op.Name] = op
	}
	return out
}()
