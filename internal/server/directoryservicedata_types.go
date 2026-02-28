package server

type directoryServiceDataAPIType struct {
	Name string
}

// AWS Directory Service Data data types sourced from:
// https://docs.aws.amazon.com/directoryservicedata/latest/DirectoryServiceDataAPIReference/API_Types.html
var directoryServiceDataAPITypes = []directoryServiceDataAPIType{
	{Name: "AccessDeniedException"},
	{Name: "AddGroupMemberRequest"},
	{Name: "AddGroupMemberResult"},
	{Name: "AttributeValue"},
	{Name: "ConflictException"},
	{Name: "CreateGroupRequest"},
	{Name: "CreateGroupResult"},
	{Name: "CreateUserRequest"},
	{Name: "CreateUserResult"},
	{Name: "DeleteGroupRequest"},
	{Name: "DeleteGroupResult"},
	{Name: "DeleteUserRequest"},
	{Name: "DeleteUserResult"},
	{Name: "DescribeGroupRequest"},
	{Name: "DescribeGroupResult"},
	{Name: "DescribeUserRequest"},
	{Name: "DescribeUserResult"},
	{Name: "DirectoryUnavailableException"},
	{Name: "DisableUserRequest"},
	{Name: "DisableUserResult"},
	{Name: "Group"},
	{Name: "GroupSummary"},
	{Name: "InternalServerException"},
	{Name: "ListGroupMembersRequest"},
	{Name: "ListGroupMembersResult"},
	{Name: "ListGroupsForMemberRequest"},
	{Name: "ListGroupsForMemberResult"},
	{Name: "ListGroupsRequest"},
	{Name: "ListGroupsResult"},
	{Name: "ListUsersRequest"},
	{Name: "ListUsersResult"},
	{Name: "Member"},
	{Name: "RemoveGroupMemberRequest"},
	{Name: "RemoveGroupMemberResult"},
	{Name: "ResourceNotFoundException"},
	{Name: "SearchGroupsRequest"},
	{Name: "SearchGroupsResult"},
	{Name: "SearchUsersRequest"},
	{Name: "SearchUsersResult"},
	{Name: "ThrottlingException"},
	{Name: "UpdateGroupRequest"},
	{Name: "UpdateGroupResult"},
	{Name: "UpdateUserRequest"},
	{Name: "UpdateUserResult"},
	{Name: "User"},
	{Name: "UserSummary"},
	{Name: "ValidationException"},
}

var directoryServiceDataAPITypeByName = func() map[string]directoryServiceDataAPIType {
	out := make(map[string]directoryServiceDataAPIType, len(directoryServiceDataAPITypes))
	for _, dt := range directoryServiceDataAPITypes {
		out[dt.Name] = dt
	}
	return out
}()
