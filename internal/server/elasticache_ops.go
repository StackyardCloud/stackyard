package server

type elastiCacheOperation struct {
	Name string
}

// Amazon ElastiCache actions sourced from:
// https://docs.aws.amazon.com/AmazonElastiCache/latest/APIReference/API_Operations.html
var elastiCacheOperations = []elastiCacheOperation{
	{Name: "AddTagsToResource"},
	{Name: "AuthorizeCacheSecurityGroupIngress"},
	{Name: "BatchApplyUpdateAction"},
	{Name: "BatchStopUpdateAction"},
	{Name: "CompleteMigration"},
	{Name: "CopyServerlessCacheSnapshot"},
	{Name: "CopySnapshot"},
	{Name: "CreateCacheCluster"},
	{Name: "CreateCacheParameterGroup"},
	{Name: "CreateCacheSecurityGroup"},
	{Name: "CreateCacheSubnetGroup"},
	{Name: "CreateGlobalReplicationGroup"},
	{Name: "CreateReplicationGroup"},
	{Name: "CreateServerlessCache"},
	{Name: "CreateServerlessCacheSnapshot"},
	{Name: "CreateSnapshot"},
	{Name: "CreateUser"},
	{Name: "CreateUserGroup"},
	{Name: "DecreaseNodeGroupsInGlobalReplicationGroup"},
	{Name: "DecreaseReplicaCount"},
	{Name: "DeleteCacheCluster"},
	{Name: "DeleteCacheParameterGroup"},
	{Name: "DeleteCacheSecurityGroup"},
	{Name: "DeleteCacheSubnetGroup"},
	{Name: "DeleteGlobalReplicationGroup"},
	{Name: "DeleteReplicationGroup"},
	{Name: "DeleteServerlessCache"},
	{Name: "DeleteServerlessCacheSnapshot"},
	{Name: "DeleteSnapshot"},
	{Name: "DeleteUser"},
	{Name: "DeleteUserGroup"},
	{Name: "DescribeCacheClusters"},
	{Name: "DescribeCacheEngineVersions"},
	{Name: "DescribeCacheParameterGroups"},
	{Name: "DescribeCacheParameters"},
	{Name: "DescribeCacheSecurityGroups"},
	{Name: "DescribeCacheSubnetGroups"},
	{Name: "DescribeEngineDefaultParameters"},
	{Name: "DescribeEvents"},
	{Name: "DescribeGlobalReplicationGroups"},
	{Name: "DescribeReplicationGroups"},
	{Name: "DescribeReservedCacheNodes"},
	{Name: "DescribeReservedCacheNodesOfferings"},
	{Name: "DescribeServerlessCacheSnapshots"},
	{Name: "DescribeServerlessCaches"},
	{Name: "DescribeServiceUpdates"},
	{Name: "DescribeSnapshots"},
	{Name: "DescribeUpdateActions"},
	{Name: "DescribeUserGroups"},
	{Name: "DescribeUsers"},
	{Name: "DisassociateGlobalReplicationGroup"},
	{Name: "ExportServerlessCacheSnapshot"},
	{Name: "FailoverGlobalReplicationGroup"},
	{Name: "IncreaseNodeGroupsInGlobalReplicationGroup"},
	{Name: "IncreaseReplicaCount"},
	{Name: "ListAllowedNodeTypeModifications"},
	{Name: "ListTagsForResource"},
	{Name: "ModifyCacheCluster"},
	{Name: "ModifyCacheParameterGroup"},
	{Name: "ModifyCacheSubnetGroup"},
	{Name: "ModifyGlobalReplicationGroup"},
	{Name: "ModifyReplicationGroup"},
	{Name: "ModifyReplicationGroupShardConfiguration"},
	{Name: "ModifyServerlessCache"},
	{Name: "ModifyUser"},
	{Name: "ModifyUserGroup"},
	{Name: "PurchaseReservedCacheNodesOffering"},
	{Name: "RebalanceSlotsInGlobalReplicationGroup"},
	{Name: "RebootCacheCluster"},
	{Name: "RemoveTagsFromResource"},
	{Name: "ResetCacheParameterGroup"},
	{Name: "RevokeCacheSecurityGroupIngress"},
	{Name: "StartMigration"},
	{Name: "TestFailover"},
	{Name: "TestMigration"},
}

var elastiCacheOperationByName = func() map[string]elastiCacheOperation {
	out := make(map[string]elastiCacheOperation, len(elastiCacheOperations))
	for _, op := range elastiCacheOperations {
		out[op.Name] = op
	}
	return out
}()
