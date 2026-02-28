package server

type directoryServiceDataOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Directory Service Data operations sourced from:
// https://docs.aws.amazon.com/directoryservicedata/latest/DirectoryServiceDataAPIReference/API_Operations.html
var directoryServiceDataOperations = []directoryServiceDataOperation{
	{Name: "AddGroupMember", Method: "POST", URI: "/GroupMemberships/AddGroupMember"},
	{Name: "CreateGroup", Method: "POST", URI: "/Groups/CreateGroup"},
	{Name: "CreateUser", Method: "POST", URI: "/Users/CreateUser"},
	{Name: "DeleteGroup", Method: "POST", URI: "/Groups/DeleteGroup"},
	{Name: "DeleteUser", Method: "POST", URI: "/Users/DeleteUser"},
	{Name: "DescribeGroup", Method: "POST", URI: "/Groups/DescribeGroup"},
	{Name: "DescribeUser", Method: "POST", URI: "/Users/DescribeUser"},
	{Name: "DisableUser", Method: "POST", URI: "/Users/DisableUser"},
	{Name: "ListGroupMembers", Method: "POST", URI: "/GroupMemberships/ListGroupMembers"},
	{Name: "ListGroups", Method: "POST", URI: "/Groups/ListGroups"},
	{Name: "ListGroupsForMember", Method: "POST", URI: "/GroupMemberships/ListGroupsForMember"},
	{Name: "ListUsers", Method: "POST", URI: "/Users/ListUsers"},
	{Name: "RemoveGroupMember", Method: "POST", URI: "/GroupMemberships/RemoveGroupMember"},
	{Name: "SearchGroups", Method: "POST", URI: "/Groups/SearchGroups"},
	{Name: "SearchUsers", Method: "POST", URI: "/Users/SearchUsers"},
	{Name: "UpdateGroup", Method: "POST", URI: "/Groups/UpdateGroup"},
	{Name: "UpdateUser", Method: "POST", URI: "/Users/UpdateUser"},
}

var directoryServiceDataOperationByName = func() map[string]directoryServiceDataOperation {
	out := make(map[string]directoryServiceDataOperation, len(directoryServiceDataOperations))
	for _, op := range directoryServiceDataOperations {
		out[op.Name] = op
	}
	return out
}()
