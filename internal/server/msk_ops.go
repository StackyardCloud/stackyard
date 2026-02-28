package server

type mskOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon MSK API v2 operations sourced from:
// https://docs.aws.amazon.com/MSK/2.0/APIReference/operations.html
var mskOperations = []mskOperation{
	{Name: "CreateClusterV2", Method: "POST", URI: "/api/v2/clusters"},
	{Name: "DescribeClusterOperationV2", Method: "GET", URI: "/api/v2/operations/{clusterOperationArn}"},
	{Name: "DescribeClusterV2", Method: "GET", URI: "/api/v2/clusters/{clusterArn}"},
	{Name: "ListClusterOperationsV2", Method: "GET", URI: "/api/v2/clusters/{clusterArn}/operations"},
	{Name: "ListClustersV2", Method: "GET", URI: "/api/v2/clusters"},
}

var mskOperationByName = func() map[string]mskOperation {
	out := make(map[string]mskOperation, len(mskOperations))
	for _, op := range mskOperations {
		out[op.Name] = op
	}
	return out
}()
