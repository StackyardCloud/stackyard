package server

type daxOperation struct{ Name string }

// Amazon DynamoDB Accelerator (DAX) control-plane operations sourced from:
// https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_Operations_Amazon_DynamoDB_Accelerator.html
var daxOperations = []daxOperation{
	{Name: "CreateCluster"},
	{Name: "CreateParameterGroup"},
	{Name: "CreateSubnetGroup"},
	{Name: "DecreaseReplicationFactor"},
	{Name: "DeleteCluster"},
	{Name: "DeleteParameterGroup"},
	{Name: "DeleteSubnetGroup"},
	{Name: "DescribeClusters"},
	{Name: "DescribeDefaultParameters"},
	{Name: "DescribeEvents"},
	{Name: "DescribeParameterGroups"},
	{Name: "DescribeParameters"},
	{Name: "DescribeSubnetGroups"},
	{Name: "IncreaseReplicationFactor"},
	{Name: "ListTags"},
	{Name: "RebootNode"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateCluster"},
	{Name: "UpdateParameterGroup"},
	{Name: "UpdateSubnetGroup"},
}

var daxOperationByName = func() map[string]daxOperation {
	out := make(map[string]daxOperation, len(daxOperations))
	for _, op := range daxOperations {
		out[op.Name] = op
	}
	return out
}()
