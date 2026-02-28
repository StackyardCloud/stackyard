package server

type resourceGroupsDataType struct {
	Name string
}

// AWS Resource Groups data types sourced from:
// https://docs.aws.amazon.com/ARG/latest/APIReference/API_Types.html
var resourceGroupsDataTypes = []resourceGroupsDataType{
	{Name: "AccountSettings"},
	{Name: "FailedResource"},
	{Name: "Group"},
	{Name: "GroupConfiguration"},
	{Name: "GroupConfigurationItem"},
	{Name: "GroupConfigurationParameter"},
	{Name: "GroupFilter"},
	{Name: "GroupIdentifier"},
	{Name: "GroupQuery"},
	{Name: "GroupingStatusesItem"},
	{Name: "ListGroupResourcesItem"},
	{Name: "ListGroupingStatusesFilter"},
	{Name: "ListTagSyncTasksFilter"},
	{Name: "PendingResource"},
	{Name: "QueryError"},
	{Name: "ResourceFilter"},
	{Name: "ResourceIdentifier"},
	{Name: "ResourceQuery"},
	{Name: "ResourceStatus"},
	{Name: "TagSyncTaskItem"},
	{Name: "UpdateGroupQuery"},
}

var resourceGroupsDataTypeByName = func() map[string]resourceGroupsDataType {
	out := make(map[string]resourceGroupsDataType, len(resourceGroupsDataTypes))
	for _, dt := range resourceGroupsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
