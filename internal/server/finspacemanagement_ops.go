package server

type finspaceManagementOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon FinSpace Management API actions sourced from:
// https://docs.aws.amazon.com/finspace/latest/management-api/API_Operations.html
var finspaceManagementOperations = []finspaceManagementOperation{
	{Name: "CreateEnvironment", Method: "POST", URI: "/environment"},
	{Name: "CreateKxChangeset", Method: "POST", URI: "/kx/environments/{environmentId}/databases/{databaseName}/changesets"},
	{Name: "CreateKxCluster", Method: "POST", URI: "/kx/environments/{environmentId}/clusters"},
	{Name: "CreateKxDatabase", Method: "POST", URI: "/kx/environments/{environmentId}/databases"},
	{Name: "CreateKxDataview", Method: "POST", URI: "/kx/environments/{environmentId}/databases/{databaseName}/dataviews"},
	{Name: "CreateKxEnvironment", Method: "POST", URI: "/kx/environments"},
	{Name: "CreateKxScalingGroup", Method: "POST", URI: "/kx/environments/{environmentId}/scalingGroups"},
	{Name: "CreateKxUser", Method: "POST", URI: "/kx/environments/{environmentId}/users"},
	{Name: "CreateKxVolume", Method: "POST", URI: "/kx/environments/{environmentId}/kxvolumes"},
	{Name: "DeleteEnvironment", Method: "DELETE", URI: "/environment/{environmentId}"},
	{Name: "DeleteKxCluster", Method: "DELETE", URI: "/kx/environments/{environmentId}/clusters/{clusterName}"},
	{Name: "DeleteKxClusterNode", Method: "DELETE", URI: "/kx/environments/{environmentId}/clusters/{clusterName}/nodes/{nodeId}"},
	{Name: "DeleteKxDatabase", Method: "DELETE", URI: "/kx/environments/{environmentId}/databases/{databaseName}"},
	{Name: "DeleteKxDataview", Method: "DELETE", URI: "/kx/environments/{environmentId}/databases/{databaseName}/dataviews/{dataviewName}"},
	{Name: "DeleteKxEnvironment", Method: "DELETE", URI: "/kx/environments/{environmentId}"},
	{Name: "DeleteKxScalingGroup", Method: "DELETE", URI: "/kx/environments/{environmentId}/scalingGroups/{scalingGroupName}"},
	{Name: "DeleteKxUser", Method: "DELETE", URI: "/kx/environments/{environmentId}/users/{userName}"},
	{Name: "DeleteKxVolume", Method: "DELETE", URI: "/kx/environments/{environmentId}/kxvolumes/{volumeName}"},
	{Name: "GetEnvironment", Method: "GET", URI: "/environment/{environmentId}"},
	{Name: "GetKxChangeset", Method: "GET", URI: "/kx/environments/{environmentId}/databases/{databaseName}/changesets/{changesetId}"},
	{Name: "GetKxCluster", Method: "GET", URI: "/kx/environments/{environmentId}/clusters/{clusterName}"},
	{Name: "GetKxConnectionString", Method: "GET", URI: "/kx/environments/{environmentId}/connectionString"},
	{Name: "GetKxDatabase", Method: "GET", URI: "/kx/environments/{environmentId}/databases/{databaseName}"},
	{Name: "GetKxDataview", Method: "GET", URI: "/kx/environments/{environmentId}/databases/{databaseName}/dataviews/{dataviewName}"},
	{Name: "GetKxEnvironment", Method: "GET", URI: "/kx/environments/{environmentId}"},
	{Name: "GetKxScalingGroup", Method: "GET", URI: "/kx/environments/{environmentId}/scalingGroups/{scalingGroupName}"},
	{Name: "GetKxUser", Method: "GET", URI: "/kx/environments/{environmentId}/users/{userName}"},
	{Name: "GetKxVolume", Method: "GET", URI: "/kx/environments/{environmentId}/kxvolumes/{volumeName}"},
	{Name: "ListEnvironments", Method: "GET", URI: "/environment"},
	{Name: "ListKxChangesets", Method: "GET", URI: "/kx/environments/{environmentId}/databases/{databaseName}/changesets"},
	{Name: "ListKxClusterNodes", Method: "GET", URI: "/kx/environments/{environmentId}/clusters/{clusterName}/nodes"},
	{Name: "ListKxClusters", Method: "GET", URI: "/kx/environments/{environmentId}/clusters"},
	{Name: "ListKxDatabases", Method: "GET", URI: "/kx/environments/{environmentId}/databases"},
	{Name: "ListKxDataviews", Method: "GET", URI: "/kx/environments/{environmentId}/databases/{databaseName}/dataviews"},
	{Name: "ListKxEnvironments", Method: "GET", URI: "/kx/environments"},
	{Name: "ListKxScalingGroups", Method: "GET", URI: "/kx/environments/{environmentId}/scalingGroups"},
	{Name: "ListKxUsers", Method: "GET", URI: "/kx/environments/{environmentId}/users"},
	{Name: "ListKxVolumes", Method: "GET", URI: "/kx/environments/{environmentId}/kxvolumes"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateEnvironment", Method: "PUT", URI: "/environment/{environmentId}"},
	{Name: "UpdateKxClusterCodeConfiguration", Method: "PUT", URI: "/kx/environments/{environmentId}/clusters/{clusterName}/configuration/code"},
	{Name: "UpdateKxClusterDatabases", Method: "PUT", URI: "/kx/environments/{environmentId}/clusters/{clusterName}/configuration/databases"},
	{Name: "UpdateKxDatabase", Method: "PUT", URI: "/kx/environments/{environmentId}/databases/{databaseName}"},
	{Name: "UpdateKxDataview", Method: "PUT", URI: "/kx/environments/{environmentId}/databases/{databaseName}/dataviews/{dataviewName}"},
	{Name: "UpdateKxEnvironment", Method: "PUT", URI: "/kx/environments/{environmentId}"},
	{Name: "UpdateKxEnvironmentNetwork", Method: "PUT", URI: "/kx/environments/{environmentId}/network"},
	{Name: "UpdateKxUser", Method: "PUT", URI: "/kx/environments/{environmentId}/users/{userName}"},
	{Name: "UpdateKxVolume", Method: "PATCH", URI: "/kx/environments/{environmentId}/kxvolumes/{volumeName}"},
}

var finspaceManagementOperationByName = func() map[string]finspaceManagementOperation {
	out := make(map[string]finspaceManagementOperation, len(finspaceManagementOperations))
	for _, op := range finspaceManagementOperations {
		out[op.Name] = op
	}
	return out
}()
