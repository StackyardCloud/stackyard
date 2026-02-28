package server

type memorydbOperation struct {
	Name string
}

// Amazon MemoryDB operations sourced from:
// https://docs.aws.amazon.com/memorydb/latest/APIReference/API_Operations.html
var memorydbOperations = []memorydbOperation{
	{Name: "BatchUpdateCluster"},
	{Name: "CopySnapshot"},
	{Name: "CreateACL"},
	{Name: "CreateCluster"},
	{Name: "CreateMultiRegionCluster"},
	{Name: "CreateParameterGroup"},
	{Name: "CreateSnapshot"},
	{Name: "CreateSubnetGroup"},
	{Name: "CreateUser"},
	{Name: "DeleteACL"},
	{Name: "DeleteCluster"},
	{Name: "DeleteMultiRegionCluster"},
	{Name: "DeleteParameterGroup"},
	{Name: "DeleteSnapshot"},
	{Name: "DeleteSubnetGroup"},
	{Name: "DeleteUser"},
	{Name: "DescribeACLs"},
	{Name: "DescribeClusters"},
	{Name: "DescribeEngineVersions"},
	{Name: "DescribeEvents"},
	{Name: "DescribeMultiRegionClusters"},
	{Name: "DescribeParameterGroups"},
	{Name: "DescribeParameters"},
	{Name: "DescribeReservedNodes"},
	{Name: "DescribeReservedNodesOfferings"},
	{Name: "DescribeServiceUpdates"},
	{Name: "DescribeSnapshots"},
	{Name: "DescribeSubnetGroups"},
	{Name: "DescribeUsers"},
	{Name: "FailoverShard"},
	{Name: "ListAllowedMultiRegionClusterUpdates"},
	{Name: "ListAllowedNodeTypeUpdates"},
	{Name: "ListTags"},
	{Name: "PurchaseReservedNodesOffering"},
	{Name: "ResetParameterGroup"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateACL"},
	{Name: "UpdateCluster"},
	{Name: "UpdateMultiRegionCluster"},
	{Name: "UpdateParameterGroup"},
	{Name: "UpdateSubnetGroup"},
	{Name: "UpdateUser"},
}

var memorydbOperationByName = func() map[string]memorydbOperation {
	out := make(map[string]memorydbOperation, len(memorydbOperations))
	for _, op := range memorydbOperations {
		out[op.Name] = op
	}
	return out
}()
