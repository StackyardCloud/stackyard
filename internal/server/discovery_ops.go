package server

type discoveryOperation struct {
	Name string
}

// AWS Application Discovery Service actions sourced from:
// https://docs.aws.amazon.com/application-discovery/latest/APIReference/API_Operations.html
var discoveryOperations = []discoveryOperation{
	{Name: "AssociateConfigurationItemsToApplication"},
	{Name: "BatchDeleteAgents"},
	{Name: "BatchDeleteImportData"},
	{Name: "CreateApplication"},
	{Name: "CreateTags"},
	{Name: "DeleteApplications"},
	{Name: "DeleteTags"},
	{Name: "DescribeAgents"},
	{Name: "DescribeBatchDeleteConfigurationTask"},
	{Name: "DescribeConfigurations"},
	{Name: "DescribeContinuousExports"},
	{Name: "DescribeExportConfigurations"},
	{Name: "DescribeExportTasks"},
	{Name: "DescribeImportTasks"},
	{Name: "DescribeTags"},
	{Name: "DisassociateConfigurationItemsFromApplication"},
	{Name: "ExportConfigurations"},
	{Name: "GetDiscoverySummary"},
	{Name: "ListConfigurations"},
	{Name: "ListServerNeighbors"},
	{Name: "StartBatchDeleteConfigurationTask"},
	{Name: "StartContinuousExport"},
	{Name: "StartDataCollectionByAgentIds"},
	{Name: "StartExportTask"},
	{Name: "StartImportTask"},
	{Name: "StopContinuousExport"},
	{Name: "StopDataCollectionByAgentIds"},
	{Name: "UpdateApplication"},
}

var discoveryOperationByName = func() map[string]discoveryOperation {
	out := make(map[string]discoveryOperation, len(discoveryOperations))
	for _, op := range discoveryOperations {
		out[op.Name] = op
	}
	return out
}()
