package server

type identityStoreOperation struct {
	Name string
}

// AWS Identity Store operations sourced from:
// https://docs.aws.amazon.com/singlesignon/latest/IdentityStoreAPIReference/API_Operations.html
var identityStoreOperations = []identityStoreOperation{
	{Name: "CreateGroup"},
	{Name: "CreateGroupMembership"},
	{Name: "CreateUser"},
	{Name: "DeleteGroup"},
	{Name: "DeleteGroupMembership"},
	{Name: "DeleteUser"},
	{Name: "DescribeGroup"},
	{Name: "DescribeGroupMembership"},
	{Name: "DescribeUser"},
	{Name: "GetGroupId"},
	{Name: "GetGroupMembershipId"},
	{Name: "GetUserId"},
	{Name: "IsMemberInGroups"},
	{Name: "ListGroupMemberships"},
	{Name: "ListGroupMembershipsForMember"},
	{Name: "ListGroups"},
	{Name: "ListUsers"},
	{Name: "UpdateGroup"},
	{Name: "UpdateUser"},
}

var identityStoreOperationByName = func() map[string]identityStoreOperation {
	out := make(map[string]identityStoreOperation, len(identityStoreOperations))
	for _, op := range identityStoreOperations {
		out[op.Name] = op
	}
	return out
}()
