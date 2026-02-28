package server

type keyspacesOperation struct {
	Name string
}

// Amazon Keyspaces operations sourced from:
// https://docs.aws.amazon.com/keyspaces/latest/APIReference/API_Operations.html
var keyspacesOperations = []keyspacesOperation{
	{Name: "CreateKeyspace"},
	{Name: "CreateTable"},
	{Name: "CreateType"},
	{Name: "DeleteKeyspace"},
	{Name: "DeleteTable"},
	{Name: "DeleteType"},
	{Name: "GetKeyspace"},
	{Name: "GetTable"},
	{Name: "GetTableAutoScalingSettings"},
	{Name: "GetType"},
	{Name: "ListKeyspaces"},
	{Name: "ListTables"},
	{Name: "ListTagsForResource"},
	{Name: "ListTypes"},
	{Name: "RestoreTable"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateKeyspace"},
	{Name: "UpdateTable"},
}

var keyspacesOperationByName = func() map[string]keyspacesOperation {
	out := make(map[string]keyspacesOperation, len(keyspacesOperations))
	for _, op := range keyspacesOperations {
		out[op.Name] = op
	}
	return out
}()
