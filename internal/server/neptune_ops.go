package server

type neptuneOperation struct {
	Name string
}

// Amazon Neptune operations sourced from:
// https://docs.aws.amazon.com/neptune/latest/apiref/API_Operations.html
var neptuneOperations = []neptuneOperation{
	{Name: "AddRoleToDBCluster"},
	{Name: "AddSourceIdentifierToSubscription"},
	{Name: "AddTagsToResource"},
	{Name: "ApplyPendingMaintenanceAction"},
	{Name: "CopyDBClusterParameterGroup"},
	{Name: "CopyDBClusterSnapshot"},
	{Name: "CopyDBParameterGroup"},
	{Name: "CreateDBCluster"},
	{Name: "CreateDBClusterEndpoint"},
	{Name: "CreateDBClusterParameterGroup"},
	{Name: "CreateDBClusterSnapshot"},
	{Name: "CreateDBInstance"},
	{Name: "CreateDBParameterGroup"},
	{Name: "CreateDBSubnetGroup"},
	{Name: "CreateEventSubscription"},
	{Name: "CreateGlobalCluster"},
	{Name: "DeleteDBCluster"},
	{Name: "DeleteDBClusterEndpoint"},
	{Name: "DeleteDBClusterParameterGroup"},
	{Name: "DeleteDBClusterSnapshot"},
	{Name: "DeleteDBInstance"},
	{Name: "DeleteDBParameterGroup"},
	{Name: "DeleteDBSubnetGroup"},
	{Name: "DeleteEventSubscription"},
	{Name: "DeleteGlobalCluster"},
	{Name: "DescribeDBClusterEndpoints"},
	{Name: "DescribeDBClusterParameterGroups"},
	{Name: "DescribeDBClusterParameters"},
	{Name: "DescribeDBClusterSnapshotAttributes"},
	{Name: "DescribeDBClusterSnapshots"},
	{Name: "DescribeDBClusters"},
	{Name: "DescribeDBEngineVersions"},
	{Name: "DescribeDBInstances"},
	{Name: "DescribeDBParameterGroups"},
	{Name: "DescribeDBParameters"},
	{Name: "DescribeDBSubnetGroups"},
	{Name: "DescribeEngineDefaultClusterParameters"},
	{Name: "DescribeEngineDefaultParameters"},
	{Name: "DescribeEventCategories"},
	{Name: "DescribeEventSubscriptions"},
	{Name: "DescribeEvents"},
	{Name: "DescribeGlobalClusters"},
	{Name: "DescribeOrderableDBInstanceOptions"},
	{Name: "DescribePendingMaintenanceActions"},
	{Name: "DescribeValidDBInstanceModifications"},
	{Name: "FailoverDBCluster"},
	{Name: "FailoverGlobalCluster"},
	{Name: "ListTagsForResource"},
	{Name: "ModifyDBCluster"},
	{Name: "ModifyDBClusterEndpoint"},
	{Name: "ModifyDBClusterParameterGroup"},
	{Name: "ModifyDBClusterSnapshotAttribute"},
	{Name: "ModifyDBInstance"},
	{Name: "ModifyDBParameterGroup"},
	{Name: "ModifyDBSubnetGroup"},
	{Name: "ModifyEventSubscription"},
	{Name: "ModifyGlobalCluster"},
	{Name: "PromoteReadReplicaDBCluster"},
	{Name: "RebootDBInstance"},
	{Name: "RemoveFromGlobalCluster"},
	{Name: "RemoveRoleFromDBCluster"},
	{Name: "RemoveSourceIdentifierFromSubscription"},
	{Name: "RemoveTagsFromResource"},
	{Name: "ResetDBClusterParameterGroup"},
	{Name: "ResetDBParameterGroup"},
	{Name: "RestoreDBClusterFromSnapshot"},
	{Name: "RestoreDBClusterToPointInTime"},
	{Name: "StartDBCluster"},
	{Name: "StopDBCluster"},
	{Name: "SwitchoverGlobalCluster"},
}

var neptuneOperationByName = func() map[string]neptuneOperation {
	out := make(map[string]neptuneOperation, len(neptuneOperations))
	for _, op := range neptuneOperations {
		out[op.Name] = op
	}
	return out
}()
