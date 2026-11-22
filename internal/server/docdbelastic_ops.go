package server

type docdbelasticOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon DocumentDB Elastic Clusters operations sourced from:
// https://docs.aws.amazon.com/documentdb/latest/developerguide/API_Operations_Amazon_DocumentDB_Elastic_Clusters.html
var docdbelasticOperations = []docdbelasticOperation{
	{Name: "ApplyPendingMaintenanceAction", Method: "POST", URI: "/pending-action"},
	{Name: "CopyClusterSnapshot", Method: "POST", URI: "/cluster-snapshot/{snapshotArn}/copy"},
	{Name: "CreateCluster", Method: "POST", URI: "/cluster"},
	{Name: "CreateClusterSnapshot", Method: "POST", URI: "/cluster-snapshot"},
	{Name: "DeleteCluster", Method: "DELETE", URI: "/cluster/{clusterArn}"},
	{Name: "DeleteClusterSnapshot", Method: "DELETE", URI: "/cluster-snapshot/{snapshotArn}"},
	{Name: "GetCluster", Method: "GET", URI: "/cluster/{clusterArn}"},
	{Name: "GetClusterSnapshot", Method: "GET", URI: "/cluster-snapshot/{snapshotArn}"},
	{Name: "GetPendingMaintenanceAction", Method: "GET", URI: "/pending-action/{resourceArn}"},
	{Name: "ListClusterSnapshots", Method: "GET", URI: "/cluster-snapshots"},
	{Name: "ListClusters", Method: "GET", URI: "/clusters"},
	{Name: "ListPendingMaintenanceActions", Method: "GET", URI: "/pending-actions"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "RestoreClusterFromSnapshot", Method: "POST", URI: "/cluster-snapshot/{snapshotArn}/restore"},
	{Name: "StartCluster", Method: "POST", URI: "/cluster/{clusterArn}/start"},
	{Name: "StopCluster", Method: "POST", URI: "/cluster/{clusterArn}/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateCluster", Method: "PUT", URI: "/cluster/{clusterArn}"},
}

var docdbelasticOperationByName = func() map[string]docdbelasticOperation {
	out := make(map[string]docdbelasticOperation, len(docdbelasticOperations))
	for _, op := range docdbelasticOperations {
		out[op.Name] = op
	}
	return out
}()
