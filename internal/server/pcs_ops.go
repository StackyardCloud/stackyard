package server

type pcsOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS PCS operations sourced from:
// https://docs.aws.amazon.com/pcs/latest/APIReference/API_Operations.html
var pcsOperations = []pcsOperation{
	{Name: "CreateCluster", Method: "POST", URI: "/"},
	{Name: "CreateComputeNodeGroup", Method: "POST", URI: "/"},
	{Name: "CreateQueue", Method: "POST", URI: "/"},
	{Name: "DeleteCluster", Method: "POST", URI: "/"},
	{Name: "DeleteComputeNodeGroup", Method: "POST", URI: "/"},
	{Name: "DeleteQueue", Method: "POST", URI: "/"},
	{Name: "GetCluster", Method: "POST", URI: "/"},
	{Name: "GetComputeNodeGroup", Method: "POST", URI: "/"},
	{Name: "GetQueue", Method: "POST", URI: "/"},
	{Name: "ListClusters", Method: "POST", URI: "/"},
	{Name: "ListComputeNodeGroups", Method: "POST", URI: "/"},
	{Name: "ListQueues", Method: "POST", URI: "/"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/"},
	{Name: "RegisterComputeNodeGroupInstance", Method: "POST", URI: "/"},
	{Name: "TagResource", Method: "POST", URI: "/"},
	{Name: "UntagResource", Method: "POST", URI: "/"},
	{Name: "UpdateCluster", Method: "POST", URI: "/"},
	{Name: "UpdateComputeNodeGroup", Method: "POST", URI: "/"},
	{Name: "UpdateQueue", Method: "POST", URI: "/"},
}

var pcsOperationByName = func() map[string]pcsOperation {
	out := make(map[string]pcsOperation, len(pcsOperations))
	for _, op := range pcsOperations {
		out[op.Name] = op
	}
	return out
}()
