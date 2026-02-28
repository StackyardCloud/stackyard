package server

type organizationsOperation struct {
	Name string
}

// AWS Organizations operations sourced from:
// https://docs.aws.amazon.com/organizations/latest/APIReference/API_Operations.html
var organizationsOperations = []organizationsOperation{
	{Name: "AcceptHandshake"},
	{Name: "AttachPolicy"},
	{Name: "CancelHandshake"},
	{Name: "CloseAccount"},
	{Name: "CreateAccount"},
	{Name: "CreateGovCloudAccount"},
	{Name: "CreateOrganization"},
	{Name: "CreateOrganizationalUnit"},
	{Name: "CreatePolicy"},
	{Name: "DeclineHandshake"},
	{Name: "DeleteOrganization"},
	{Name: "DeleteOrganizationalUnit"},
	{Name: "DeletePolicy"},
	{Name: "DeleteResourcePolicy"},
	{Name: "DeregisterDelegatedAdministrator"},
	{Name: "DescribeAccount"},
	{Name: "DescribeCreateAccountStatus"},
	{Name: "DescribeEffectivePolicy"},
	{Name: "DescribeHandshake"},
	{Name: "DescribeOrganization"},
	{Name: "DescribeOrganizationalUnit"},
	{Name: "DescribePolicy"},
	{Name: "DescribeResourcePolicy"},
	{Name: "DescribeResponsibilityTransfer"},
	{Name: "DetachPolicy"},
	{Name: "DisableAWSServiceAccess"},
	{Name: "DisablePolicyType"},
	{Name: "EnableAllFeatures"},
	{Name: "EnableAWSServiceAccess"},
	{Name: "EnablePolicyType"},
	{Name: "InviteAccountToOrganization"},
	{Name: "InviteOrganizationToTransferResponsibility"},
	{Name: "LeaveOrganization"},
	{Name: "ListAWSServiceAccessForOrganization"},
	{Name: "ListAccounts"},
	{Name: "ListAccountsForParent"},
	{Name: "ListAccountsWithInvalidEffectivePolicy"},
	{Name: "ListChildren"},
	{Name: "ListCreateAccountStatus"},
	{Name: "ListDelegatedAdministrators"},
	{Name: "ListDelegatedServicesForAccount"},
	{Name: "ListEffectivePolicyValidationErrors"},
	{Name: "ListHandshakesForAccount"},
	{Name: "ListHandshakesForOrganization"},
	{Name: "ListInboundResponsibilityTransfers"},
	{Name: "ListOrganizationalUnitsForParent"},
	{Name: "ListOutboundResponsibilityTransfers"},
	{Name: "ListParents"},
	{Name: "ListPolicies"},
	{Name: "ListPoliciesForTarget"},
	{Name: "ListRoots"},
	{Name: "ListTagsForResource"},
	{Name: "ListTargetsForPolicy"},
	{Name: "MoveAccount"},
	{Name: "PutResourcePolicy"},
	{Name: "RegisterDelegatedAdministrator"},
	{Name: "RemoveAccountFromOrganization"},
	{Name: "TagResource"},
	{Name: "TerminateResponsibilityTransfer"},
	{Name: "UntagResource"},
	{Name: "UpdateOrganizationalUnit"},
	{Name: "UpdatePolicy"},
	{Name: "UpdateResponsibilityTransfer"},
}

var organizationsOperationByName = func() map[string]organizationsOperation {
	out := make(map[string]organizationsOperation, len(organizationsOperations))
	for _, op := range organizationsOperations {
		out[op.Name] = op
	}
	return out
}()
