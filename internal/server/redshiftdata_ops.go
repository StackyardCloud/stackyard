package server

type redshiftdataOperation struct {
	Name string
}

// Amazon Redshift Data API operations sourced from:
// https://docs.aws.amazon.com/redshift-data/latest/APIReference/API_Operations.html
var redshiftdataOperations = []redshiftdataOperation{
	{Name: "BatchExecuteStatement"},
	{Name: "CancelStatement"},
	{Name: "DescribeStatement"},
	{Name: "DescribeTable"},
	{Name: "ExecuteStatement"},
	{Name: "GetStatementResult"},
	{Name: "GetStatementResultV2"},
	{Name: "ListDatabases"},
	{Name: "ListSchemas"},
	{Name: "ListSessions"},
	{Name: "ListStatements"},
	{Name: "ListTables"},
}

var redshiftdataOperationByName = func() map[string]redshiftdataOperation {
	out := make(map[string]redshiftdataOperation, len(redshiftdataOperations))
	for _, op := range redshiftdataOperations {
		out[op.Name] = op
	}
	return out
}()
