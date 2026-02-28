package server

type flinkOperation struct {
	Name string
}

// Amazon Managed Service for Apache Flink actions sourced from:
// https://docs.aws.amazon.com/managed-flink/latest/apiv2/API_Operations.html
var flinkOperations = []flinkOperation{
	{Name: "AddApplicationCloudWatchLoggingOption"},
	{Name: "AddApplicationInput"},
	{Name: "AddApplicationInputProcessingConfiguration"},
	{Name: "AddApplicationOutput"},
	{Name: "AddApplicationReferenceDataSource"},
	{Name: "AddApplicationVpcConfiguration"},
	{Name: "CreateApplication"},
	{Name: "CreateApplicationPresignedUrl"},
	{Name: "CreateApplicationSnapshot"},
	{Name: "DeleteApplication"},
	{Name: "DeleteApplicationCloudWatchLoggingOption"},
	{Name: "DeleteApplicationInputProcessingConfiguration"},
	{Name: "DeleteApplicationOutput"},
	{Name: "DeleteApplicationReferenceDataSource"},
	{Name: "DeleteApplicationSnapshot"},
	{Name: "DeleteApplicationVpcConfiguration"},
	{Name: "DescribeApplication"},
	{Name: "DescribeApplicationOperation"},
	{Name: "DescribeApplicationSnapshot"},
	{Name: "DescribeApplicationVersion"},
	{Name: "DiscoverInputSchema"},
	{Name: "ListApplicationOperations"},
	{Name: "ListApplicationSnapshots"},
	{Name: "ListApplicationVersions"},
	{Name: "ListApplications"},
	{Name: "ListTagsForResource"},
	{Name: "RollbackApplication"},
	{Name: "StartApplication"},
	{Name: "StopApplication"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateApplication"},
	{Name: "UpdateApplicationMaintenanceConfiguration"},
}

var flinkOperationByName = func() map[string]flinkOperation {
	out := make(map[string]flinkOperation, len(flinkOperations))
	for _, op := range flinkOperations {
		out[op.Name] = op
	}
	return out
}()
