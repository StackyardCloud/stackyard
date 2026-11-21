package server

type rdsdataOperation struct {
	Name string
	Path string
}

// Amazon RDS Data API operations sourced from:
// https://docs.aws.amazon.com/rdsdataservice/latest/APIReference/API_Operations.html
var rdsdataOperations = []rdsdataOperation{
	{Name: "BatchExecuteStatement", Path: "/BatchExecute"},
	{Name: "BeginTransaction", Path: "/BeginTransaction"},
	{Name: "CommitTransaction", Path: "/CommitTransaction"},
	{Name: "ExecuteSql", Path: "/ExecuteSql"},
	{Name: "ExecuteStatement", Path: "/Execute"},
	{Name: "RollbackTransaction", Path: "/RollbackTransaction"},
}

var rdsdataOperationByPath = func() map[string]rdsdataOperation {
	out := make(map[string]rdsdataOperation, len(rdsdataOperations))
	for _, op := range rdsdataOperations {
		out[op.Path] = op
	}
	return out
}()
