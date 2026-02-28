package server

type diagnosticToolsOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Diagnostic Tools actions sourced from:
// https://docs.aws.amazon.com/diagnostic-tools/latest/APIReference/API_Operations.html
var diagnosticToolsOperations = []diagnosticToolsOperation{
	{Name: "GetExecution", Method: "POST", URI: "/"},
	{Name: "GetExecutionOutput", Method: "POST", URI: "/"},
	{Name: "GetTool", Method: "POST", URI: "/"},
	{Name: "ListExecutions", Method: "POST", URI: "/"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/"},
	{Name: "ListTools", Method: "POST", URI: "/"},
	{Name: "StartExecution", Method: "POST", URI: "/"},
	{Name: "TagResource", Method: "POST", URI: "/"},
	{Name: "UntagResource", Method: "POST", URI: "/"},
}

var diagnosticToolsOperationByName = func() map[string]diagnosticToolsOperation {
	out := make(map[string]diagnosticToolsOperation, len(diagnosticToolsOperations))
	for _, op := range diagnosticToolsOperations {
		out[op.Name] = op
	}
	return out
}()
