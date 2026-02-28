package server

type wafv2Operation struct {
	Name string
}

// AWS WAFV2 operations sourced from:
// https://docs.aws.amazon.com/waf/latest/APIReference/API_Operations.html
var wafv2Operations = []wafv2Operation{
	{Name: "AssociateWebACL"},
	{Name: "CheckCapacity"},
	{Name: "CreateAPIKey"},
	{Name: "CreateIPSet"},
	{Name: "CreateRegexPatternSet"},
	{Name: "CreateRuleGroup"},
	{Name: "CreateWebACL"},
	{Name: "DeleteAPIKey"},
	{Name: "DeleteFirewallManagerRuleGroups"},
	{Name: "DeleteIPSet"},
	{Name: "DeleteLoggingConfiguration"},
	{Name: "DeletePermissionPolicy"},
	{Name: "DeleteRegexPatternSet"},
	{Name: "DeleteRuleGroup"},
	{Name: "DeleteWebACL"},
	{Name: "DescribeAllManagedProducts"},
	{Name: "DescribeManagedProductsByVendor"},
	{Name: "DescribeManagedRuleGroup"},
	{Name: "DisassociateWebACL"},
	{Name: "GenerateMobileSdkReleaseUrl"},
	{Name: "GetDecryptedAPIKey"},
	{Name: "GetIPSet"},
	{Name: "GetLoggingConfiguration"},
	{Name: "GetManagedRuleSet"},
	{Name: "GetMobileSdkRelease"},
	{Name: "GetPermissionPolicy"},
	{Name: "GetRateBasedStatementManagedKeys"},
	{Name: "GetRegexPatternSet"},
	{Name: "GetRuleGroup"},
	{Name: "GetSampledRequests"},
	{Name: "GetTopPathStatisticsByTraffic"},
	{Name: "GetWebACL"},
	{Name: "GetWebACLForResource"},
	{Name: "ListAPIKeys"},
	{Name: "ListAvailableManagedRuleGroupVersions"},
	{Name: "ListAvailableManagedRuleGroups"},
	{Name: "ListIPSets"},
	{Name: "ListLoggingConfigurations"},
	{Name: "ListManagedRuleSets"},
	{Name: "ListMobileSdkReleases"},
	{Name: "ListRegexPatternSets"},
	{Name: "ListResourcesForWebACL"},
	{Name: "ListRuleGroups"},
	{Name: "ListTagsForResource"},
	{Name: "ListWebACLs"},
	{Name: "PutLoggingConfiguration"},
	{Name: "PutManagedRuleSetVersions"},
	{Name: "PutPermissionPolicy"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateIPSet"},
	{Name: "UpdateManagedRuleSetVersionExpiryDate"},
	{Name: "UpdateRegexPatternSet"},
	{Name: "UpdateRuleGroup"},
	{Name: "UpdateWebACL"},
}

var wafv2OperationByName = func() map[string]wafv2Operation {
	out := make(map[string]wafv2Operation, len(wafv2Operations))
	for _, op := range wafv2Operations {
		out[op.Name] = op
	}
	return out
}()
