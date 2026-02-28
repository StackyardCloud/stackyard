package server

type resourceGroupsOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Resource Groups operations sourced from:
// https://docs.aws.amazon.com/ARG/latest/APIReference/API_Operations.html
var resourceGroupsOperations = []resourceGroupsOperation{
	{Name: "CancelTagSyncTask", Method: "POST", URI: "/cancel-tag-sync-task"},
	{Name: "CreateGroup", Method: "POST", URI: "/groups"},
	{Name: "DeleteGroup", Method: "POST", URI: "/delete-group"},
	{Name: "GetAccountSettings", Method: "POST", URI: "/get-account-settings"},
	{Name: "GetGroup", Method: "POST", URI: "/get-group"},
	{Name: "GetGroupConfiguration", Method: "POST", URI: "/get-group-configuration"},
	{Name: "GetGroupQuery", Method: "POST", URI: "/get-group-query"},
	{Name: "GetTagSyncTask", Method: "POST", URI: "/get-tag-sync-task"},
	{Name: "GetTags", Method: "GET", URI: "/resources/{Arn}/tags"},
	{Name: "GroupResources", Method: "POST", URI: "/group-resources"},
	{Name: "ListGroupResources", Method: "POST", URI: "/list-group-resources"},
	{Name: "ListGroupingStatuses", Method: "POST", URI: "/list-grouping-statuses"},
	{Name: "ListGroups", Method: "POST", URI: "/groups-list"},
	{Name: "ListTagSyncTasks", Method: "POST", URI: "/list-tag-sync-tasks"},
	{Name: "PutGroupConfiguration", Method: "POST", URI: "/put-group-configuration"},
	{Name: "SearchResources", Method: "POST", URI: "/resources/search"},
	{Name: "StartTagSyncTask", Method: "POST", URI: "/start-tag-sync-task"},
	{Name: "Tag", Method: "PUT", URI: "/resources/{Arn}/tags"},
	{Name: "UngroupResources", Method: "POST", URI: "/ungroup-resources"},
	{Name: "Untag", Method: "PATCH", URI: "/resources/{Arn}/tags"},
	{Name: "UpdateAccountSettings", Method: "POST", URI: "/update-account-settings"},
	{Name: "UpdateGroup", Method: "POST", URI: "/update-group"},
	{Name: "UpdateGroupQuery", Method: "POST", URI: "/update-group-query"},
}

var resourceGroupsOperationByName = func() map[string]resourceGroupsOperation {
	out := make(map[string]resourceGroupsOperation, len(resourceGroupsOperations))
	for _, op := range resourceGroupsOperations {
		out[op.Name] = op
	}
	return out
}()
