package server

type cloudTrailOperation struct {
	Name string
}

// AWS CloudTrail operations sourced from:
// https://docs.aws.amazon.com/awscloudtrail/latest/APIReference/API_Operations.html
var cloudTrailOperations = []cloudTrailOperation{
	{Name: "AddTags"},
	{Name: "CancelQuery"},
	{Name: "CreateChannel"},
	{Name: "CreateDashboard"},
	{Name: "CreateEventDataStore"},
	{Name: "CreateTrail"},
	{Name: "DeleteChannel"},
	{Name: "DeleteDashboard"},
	{Name: "DeleteEventDataStore"},
	{Name: "DeleteResourcePolicy"},
	{Name: "DeleteTrail"},
	{Name: "DeregisterOrganizationDelegatedAdmin"},
	{Name: "DescribeQuery"},
	{Name: "DescribeTrails"},
	{Name: "DisableFederation"},
	{Name: "EnableFederation"},
	{Name: "GenerateQuery"},
	{Name: "GetChannel"},
	{Name: "GetDashboard"},
	{Name: "GetEventConfiguration"},
	{Name: "GetEventDataStore"},
	{Name: "GetEventSelectors"},
	{Name: "GetImport"},
	{Name: "GetInsightSelectors"},
	{Name: "GetQueryResults"},
	{Name: "GetResourcePolicy"},
	{Name: "GetTrail"},
	{Name: "GetTrailStatus"},
	{Name: "ListChannels"},
	{Name: "ListDashboards"},
	{Name: "ListEventDataStores"},
	{Name: "ListImportFailures"},
	{Name: "ListImports"},
	{Name: "ListInsightsData"},
	{Name: "ListInsightsMetricData"},
	{Name: "ListPublicKeys"},
	{Name: "ListQueries"},
	{Name: "ListTags"},
	{Name: "ListTrails"},
	{Name: "LookupEvents"},
	{Name: "PutEventConfiguration"},
	{Name: "PutEventSelectors"},
	{Name: "PutInsightSelectors"},
	{Name: "PutResourcePolicy"},
	{Name: "RegisterOrganizationDelegatedAdmin"},
	{Name: "RemoveTags"},
	{Name: "RestoreEventDataStore"},
	{Name: "SearchSampleQueries"},
	{Name: "StartDashboardRefresh"},
	{Name: "StartEventDataStoreIngestion"},
	{Name: "StartImport"},
	{Name: "StartLogging"},
	{Name: "StartQuery"},
	{Name: "StopEventDataStoreIngestion"},
	{Name: "StopImport"},
	{Name: "StopLogging"},
	{Name: "UpdateChannel"},
	{Name: "UpdateDashboard"},
	{Name: "UpdateEventDataStore"},
	{Name: "UpdateTrail"},
}

var cloudTrailOperationByName = func() map[string]cloudTrailOperation {
	out := make(map[string]cloudTrailOperation, len(cloudTrailOperations))
	for _, op := range cloudTrailOperations {
		out[op.Name] = op
	}
	return out
}()
