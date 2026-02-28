package server

type dsqlOperation struct {
	Name string
}

// Amazon Aurora DSQL operations sourced from:
// https://docs.aws.amazon.com/aurora-dsql/latest/APIReference/API_Operations.html
var dsqlOperations = []dsqlOperation{
	{Name: "CreateCluster"},
	{Name: "DeleteCluster"},
	{Name: "DeleteClusterPolicy"},
	{Name: "GetCluster"},
	{Name: "GetClusterPolicy"},
	{Name: "GetVpcEndpointServiceName"},
	{Name: "ListClusters"},
	{Name: "ListTagsForResource"},
	{Name: "PutClusterPolicy"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateCluster"},
}

var dsqlOperationByName = func() map[string]dsqlOperation {
	out := make(map[string]dsqlOperation, len(dsqlOperations))
	for _, op := range dsqlOperations {
		out[op.Name] = op
	}
	return out
}()
