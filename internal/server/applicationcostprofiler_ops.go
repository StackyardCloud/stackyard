package server

type acpOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Application Cost Profiler operations sourced from the AWS CLI command reference
// (the service's API reference pages are no longer published):
// https://docs.aws.amazon.com/cli/latest/reference/applicationcostprofiler/index.html
var acpOperations = []acpOperation{
	{Name: "DeleteReportDefinition", Method: "DELETE", URI: "/reportDefinition/{reportId}"},
	{Name: "GetReportDefinition", Method: "GET", URI: "/reportDefinition/{reportId}"},
	{Name: "ImportApplicationUsage", Method: "POST", URI: "/importApplicationUsage"},
	{Name: "ListReportDefinitions", Method: "GET", URI: "/reportDefinition"},
	{Name: "PutReportDefinition", Method: "POST", URI: "/reportDefinition"},
	{Name: "UpdateReportDefinition", Method: "PUT", URI: "/reportDefinition/{reportId}"},
}
