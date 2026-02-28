package server

type fmsOperation struct {
	Name string
}

// AWS Firewall Manager operations sourced from:
// https://docs.aws.amazon.com/fms/2018-01-01/APIReference/API_Operations.html
var fmsOperations = []fmsOperation{
	{Name: "AssociateAdminAccount"},
	{Name: "AssociateThirdPartyFirewall"},
	{Name: "BatchAssociateResource"},
	{Name: "BatchDisassociateResource"},
	{Name: "DeleteAppsList"},
	{Name: "DeleteNotificationChannel"},
	{Name: "DeletePolicy"},
	{Name: "DeleteProtocolsList"},
	{Name: "DeleteResourceSet"},
	{Name: "DisassociateAdminAccount"},
	{Name: "DisassociateThirdPartyFirewall"},
	{Name: "GetAdminAccount"},
	{Name: "GetAdminScope"},
	{Name: "GetAppsList"},
	{Name: "GetComplianceDetail"},
	{Name: "GetNotificationChannel"},
	{Name: "GetPolicy"},
	{Name: "GetProtectionStatus"},
	{Name: "GetProtocolsList"},
	{Name: "GetResourceSet"},
	{Name: "GetThirdPartyFirewallAssociationStatus"},
	{Name: "GetViolationDetails"},
	{Name: "ListAdminAccountsForOrganization"},
	{Name: "ListAdminsManagingAccount"},
	{Name: "ListAppsLists"},
	{Name: "ListComplianceStatus"},
	{Name: "ListDiscoveredResources"},
	{Name: "ListMemberAccounts"},
	{Name: "ListPolicies"},
	{Name: "ListProtocolsLists"},
	{Name: "ListResourceSetResources"},
	{Name: "ListResourceSets"},
	{Name: "ListTagsForResource"},
	{Name: "ListThirdPartyFirewallFirewallPolicies"},
	{Name: "PutAdminAccount"},
	{Name: "PutAppsList"},
	{Name: "PutNotificationChannel"},
	{Name: "PutPolicy"},
	{Name: "PutProtocolsList"},
	{Name: "PutResourceSet"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
}

var fmsOperationByName = func() map[string]fmsOperation {
	out := make(map[string]fmsOperation, len(fmsOperations))
	for _, op := range fmsOperations {
		out[op.Name] = op
	}
	return out
}()
