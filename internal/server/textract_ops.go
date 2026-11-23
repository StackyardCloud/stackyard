package server

type textractOperation struct{ Name string }

// Amazon Textract operations sourced from:
// https://docs.aws.amazon.com/textract/latest/dg/API_Operations.html
var textractOperations = []textractOperation{
	{Name: "AnalyzeDocument"},
	{Name: "AnalyzeExpense"},
	{Name: "AnalyzeID"},
	{Name: "CreateAdapter"},
	{Name: "CreateAdapterVersion"},
	{Name: "DeleteAdapter"},
	{Name: "DeleteAdapterVersion"},
	{Name: "DetectDocumentText"},
	{Name: "GetAdapter"},
	{Name: "GetAdapterVersion"},
	{Name: "GetDocumentAnalysis"},
	{Name: "GetDocumentTextDetection"},
	{Name: "GetExpenseAnalysis"},
	{Name: "GetLendingAnalysis"},
	{Name: "GetLendingAnalysisSummary"},
	{Name: "ListAdapterVersions"},
	{Name: "ListAdapters"},
	{Name: "ListTagsForResource"},
	{Name: "StartDocumentAnalysis"},
	{Name: "StartDocumentTextDetection"},
	{Name: "StartExpenseAnalysis"},
	{Name: "StartLendingAnalysis"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateAdapter"},
}

var textractOperationByName = func() map[string]textractOperation {
	out := make(map[string]textractOperation, len(textractOperations))
	for _, op := range textractOperations {
		out[op.Name] = op
	}
	return out
}()
