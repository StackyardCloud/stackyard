package server

type bcmdOperation struct{ Name string }

// AWS Billing and Cost Management Dashboards operations sourced from:
// https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_Operations_AWS_Billing_and_Cost_Management_Dashboards.html
var bcmdOperations = []bcmdOperation{
	{Name: "CreateDashboard"},
	{Name: "CreateScheduledReport"},
	{Name: "DeleteDashboard"},
	{Name: "DeleteScheduledReport"},
	{Name: "ExecuteScheduledReport"},
	{Name: "GetDashboard"},
	{Name: "GetResourcePolicy"},
	{Name: "GetScheduledReport"},
	{Name: "ListDashboards"},
	{Name: "ListScheduledReports"},
	{Name: "ListTagsForResource"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateDashboard"},
	{Name: "UpdateScheduledReport"},
}

var bcmdOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range bcmdOperations {
		out[op.Name] = true
	}
	return out
}()
