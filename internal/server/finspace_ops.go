package server

type finspaceOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon FinSpace Data API actions sourced from:
// https://docs.aws.amazon.com/finspace/latest/data-api/API_Operations.html
var finspaceOperations = []finspaceOperation{
	{Name: "AssociateUserToPermissionGroup", Method: "POST", URI: "/permission-group/{permissionGroupId}/users/{userId}"},
	{Name: "CreateChangeset", Method: "POST", URI: "/datasets/{datasetId}/changesetsv2"},
	{Name: "CreateDataset", Method: "POST", URI: "/datasetsv2"},
	{Name: "CreateDataView", Method: "POST", URI: "/datasets/{datasetId}/dataviewsv2"},
	{Name: "CreatePermissionGroup", Method: "POST", URI: "/permission-group"},
	{Name: "CreateUser", Method: "POST", URI: "/user"},
	{Name: "DeleteDataset", Method: "DELETE", URI: "/datasetsv2/{datasetId}"},
	{Name: "DeletePermissionGroup", Method: "DELETE", URI: "/permission-group/{permissionGroupId}"},
	{Name: "DisableUser", Method: "POST", URI: "/user/{userId}/disable"},
	{Name: "DisassociateUserFromPermissionGroup", Method: "DELETE", URI: "/permission-group/{permissionGroupId}/users/{userId}"},
	{Name: "EnableUser", Method: "POST", URI: "/user/{userId}/enable"},
	{Name: "GetChangeset", Method: "GET", URI: "/datasets/{datasetId}/changesetsv2/{changesetId}"},
	{Name: "GetDataset", Method: "GET", URI: "/datasetsv2/{datasetId}"},
	{Name: "GetDataView", Method: "GET", URI: "/datasets/{datasetId}/dataviewsv2/{dataviewId}"},
	{Name: "GetExternalDataViewAccessDetails", Method: "POST", URI: "/datasets/{datasetId}/dataviewsv2/{dataviewId}/external-access-details"},
	{Name: "GetPermissionGroup", Method: "GET", URI: "/permission-group/{permissionGroupId}"},
	{Name: "GetProgrammaticAccessCredentials", Method: "GET", URI: "/credentials/programmatic"},
	{Name: "GetUser", Method: "GET", URI: "/user/{userId}"},
	{Name: "GetWorkingLocation", Method: "POST", URI: "/workingLocationV1"},
	{Name: "ListChangesets", Method: "GET", URI: "/datasets/{datasetId}/changesetsv2"},
	{Name: "ListDatasets", Method: "GET", URI: "/datasetsv2"},
	{Name: "ListDataViews", Method: "GET", URI: "/datasets/{datasetId}/dataviewsv2"},
	{Name: "ListPermissionGroups", Method: "GET", URI: "/permission-group"},
	{Name: "ListPermissionGroupsByUser", Method: "GET", URI: "/user/{userId}/permission-groups"},
	{Name: "ListUsers", Method: "GET", URI: "/user"},
	{Name: "ListUsersByPermissionGroup", Method: "GET", URI: "/permission-group/{permissionGroupId}/users"},
	{Name: "ResetUserPassword", Method: "POST", URI: "/user/{userId}/password"},
	{Name: "UpdateChangeset", Method: "PUT", URI: "/datasets/{datasetId}/changesetsv2/{changesetId}"},
	{Name: "UpdateDataset", Method: "PUT", URI: "/datasetsv2/{datasetId}"},
	{Name: "UpdatePermissionGroup", Method: "PUT", URI: "/permission-group/{permissionGroupId}"},
	{Name: "UpdateUser", Method: "PUT", URI: "/user/{userId}"},
}

var finspaceOperationByName = func() map[string]finspaceOperation {
	out := make(map[string]finspaceOperation, len(finspaceOperations))
	for _, op := range finspaceOperations {
		out[op.Name] = op
	}
	return out
}()
