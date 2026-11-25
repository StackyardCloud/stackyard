package server

type bcmdeOperation struct{ Name string }

// AWS Billing and Cost Management Data Exports operations sourced from:
// https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_Operations_AWS_Billing_and_Cost_Management_Data_Exports.html
var bcmdeOperations = []bcmdeOperation{
	{Name: "CreateExport"},
	{Name: "DeleteExport"},
	{Name: "GetExecution"},
	{Name: "GetExport"},
	{Name: "GetTable"},
	{Name: "ListExecutions"},
	{Name: "ListExports"},
	{Name: "ListTables"},
	{Name: "ListTagsForResource"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateExport"},
}

var bcmdeOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range bcmdeOperations {
		out[op.Name] = true
	}
	return out
}()
