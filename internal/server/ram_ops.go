package server

type ramOperation struct {
	Name string
}

// AWS RAM operations sourced from:
// https://docs.aws.amazon.com/ram/latest/APIReference/API_Operations.html
var ramOperations = []ramOperation{
	{Name: "AcceptResourceShareInvitation"},
	{Name: "AssociateResourceShare"},
	{Name: "AssociateResourceSharePermission"},
	{Name: "CreatePermission"},
	{Name: "CreatePermissionVersion"},
	{Name: "CreateResourceShare"},
	{Name: "DeletePermission"},
	{Name: "DeletePermissionVersion"},
	{Name: "DeleteResourceShare"},
	{Name: "DisassociateResourceShare"},
	{Name: "DisassociateResourceSharePermission"},
	{Name: "EnableSharingWithAwsOrganization"},
	{Name: "GetPermission"},
	{Name: "GetResourcePolicies"},
	{Name: "GetResourceShareAssociations"},
	{Name: "GetResourceShareInvitations"},
	{Name: "GetResourceShares"},
	{Name: "ListPendingInvitationResources"},
	{Name: "ListPermissionAssociations"},
	{Name: "ListPermissionVersions"},
	{Name: "ListPermissions"},
	{Name: "ListPrincipals"},
	{Name: "ListReplacePermissionAssociationsWork"},
	{Name: "ListResourceSharePermissions"},
	{Name: "ListResourceTypes"},
	{Name: "ListResources"},
	{Name: "ListSourceAssociations"},
	{Name: "PromotePermissionCreatedFromPolicy"},
	{Name: "PromoteResourceShareCreatedFromPolicy"},
	{Name: "RejectResourceShareInvitation"},
	{Name: "ReplacePermissionAssociations"},
	{Name: "SetDefaultPermissionVersion"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateResourceShare"},
}

var ramOperationByName = func() map[string]ramOperation {
	out := make(map[string]ramOperation, len(ramOperations))
	for _, op := range ramOperations {
		out[op.Name] = op
	}
	return out
}()
