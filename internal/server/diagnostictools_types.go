package server

type diagnosticToolsDataType struct {
	Name string
}

// AWS Diagnostic Tools data types sourced from:
// https://docs.aws.amazon.com/diagnostic-tools/latest/APIReference/API_Types.html
var diagnosticToolsDataTypes = []diagnosticToolsDataType{
	{Name: "Execution"},
	{Name: "ExecutionSummary"},
	{Name: "Tag"},
	{Name: "Tool"},
	{Name: "ToolSummary"},
	{Name: "ToolVersion"},
	{Name: "ValidationExceptionField"},
}

var diagnosticToolsDataTypeByName = func() map[string]diagnosticToolsDataType {
	out := make(map[string]diagnosticToolsDataType, len(diagnosticToolsDataTypes))
	for _, dt := range diagnosticToolsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
