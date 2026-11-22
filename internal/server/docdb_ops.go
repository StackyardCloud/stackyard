package server

type docdbOperation struct {
	Name string
}

// Amazon DocumentDB (with MongoDB compatibility) operations sourced from:
// https://docs.aws.amazon.com/documentdb/latest/developerguide/API_Operations_Amazon_DocumentDB.html
var docdbOperations = []docdbOperation{
	{Name: "AddSourceIdentifierToSubscription"},
	{Name: "AddTagsToResource"},
	{Name: "ApplyPendingMaintenanceAction"},
	{Name: "CopyDBClusterParameterGroup"},
	{Name: "CopyDBClusterSnapshot"},
	{Name: "CreateDBCluster"},
	{Name: "CreateDBClusterParameterGroup"},
	{Name: "CreateDBClusterSnapshot"},
	{Name: "CreateDBInstance"},
	{Name: "CreateDBSubnetGroup"},
	{Name: "CreateEventSubscription"},
	{Name: "CreateGlobalCluster"},
	{Name: "DeleteDBCluster"},
	{Name: "DeleteDBClusterParameterGroup"},
	{Name: "DeleteDBClusterSnapshot"},
	{Name: "DeleteDBInstance"},
	{Name: "DeleteDBSubnetGroup"},
	{Name: "DeleteEventSubscription"},
	{Name: "DeleteGlobalCluster"},
	{Name: "DescribeCertificates"},
	{Name: "DescribeDBClusterParameterGroups"},
	{Name: "DescribeDBClusterParameters"},
	{Name: "DescribeDBClusterSnapshotAttributes"},
	{Name: "DescribeDBClusterSnapshots"},
	{Name: "DescribeDBClusters"},
	{Name: "DescribeDBEngineVersions"},
	{Name: "DescribeDBInstances"},
	{Name: "DescribeDBSubnetGroups"},
	{Name: "DescribeEngineDefaultClusterParameters"},
	{Name: "DescribeEventCategories"},
	{Name: "DescribeEventSubscriptions"},
	{Name: "DescribeEvents"},
	{Name: "DescribeGlobalClusters"},
	{Name: "DescribeOrderableDBInstanceOptions"},
	{Name: "DescribePendingMaintenanceActions"},
	{Name: "DescribeServerlessV2PlatformVersions"},
	{Name: "FailoverDBCluster"},
	{Name: "FailoverGlobalCluster"},
	{Name: "ListTagsForResource"},
	{Name: "ModifyDBCluster"},
	{Name: "ModifyDBClusterParameterGroup"},
	{Name: "ModifyDBClusterSnapshotAttribute"},
	{Name: "ModifyDBInstance"},
	{Name: "ModifyDBSubnetGroup"},
	{Name: "ModifyEventSubscription"},
	{Name: "ModifyGlobalCluster"},
	{Name: "RebootDBInstance"},
	{Name: "RemoveFromGlobalCluster"},
	{Name: "RemoveSourceIdentifierFromSubscription"},
	{Name: "RemoveTagsFromResource"},
	{Name: "ResetDBClusterParameterGroup"},
	{Name: "RestoreDBClusterFromSnapshot"},
	{Name: "RestoreDBClusterToPointInTime"},
	{Name: "StartDBCluster"},
	{Name: "StopDBCluster"},
	{Name: "SwitchoverGlobalCluster"},
}

var docdbOperationByName = func() map[string]docdbOperation {
	out := make(map[string]docdbOperation, len(docdbOperations))
	for _, op := range docdbOperations {
		out[op.Name] = op
	}
	return out
}()
