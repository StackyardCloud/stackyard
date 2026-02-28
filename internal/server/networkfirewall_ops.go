package server

type networkFirewallOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Network Firewall operations sourced from:
// https://docs.aws.amazon.com/network-firewall/latest/APIReference/API_Operations.html
var networkFirewallOperations = []networkFirewallOperation{
	{Name: "AssociateFirewallPolicy", Method: "POST", URI: "/"},
	{Name: "AssociateSubnets", Method: "POST", URI: "/"},
	{Name: "CreateFirewall", Method: "POST", URI: "/"},
	{Name: "CreateFirewallPolicy", Method: "POST", URI: "/"},
	{Name: "CreateRuleGroup", Method: "POST", URI: "/"},
	{Name: "CreateTLSInspectionConfiguration", Method: "POST", URI: "/"},
	{Name: "DeleteFirewall", Method: "POST", URI: "/"},
	{Name: "DeleteFirewallPolicy", Method: "POST", URI: "/"},
	{Name: "DeleteResourcePolicy", Method: "POST", URI: "/"},
	{Name: "DeleteRuleGroup", Method: "POST", URI: "/"},
	{Name: "DeleteTLSInspectionConfiguration", Method: "POST", URI: "/"},
	{Name: "DescribeFirewall", Method: "POST", URI: "/"},
	{Name: "DescribeFirewallPolicy", Method: "POST", URI: "/"},
	{Name: "DescribeLoggingConfiguration", Method: "POST", URI: "/"},
	{Name: "DescribeResourcePolicy", Method: "POST", URI: "/"},
	{Name: "DescribeRuleGroup", Method: "POST", URI: "/"},
	{Name: "DescribeRuleGroupMetadata", Method: "POST", URI: "/"},
	{Name: "DescribeTLSInspectionConfiguration", Method: "POST", URI: "/"},
	{Name: "DisassociateSubnets", Method: "POST", URI: "/"},
	{Name: "ListFirewallPolicies", Method: "POST", URI: "/"},
	{Name: "ListFirewalls", Method: "POST", URI: "/"},
	{Name: "ListRuleGroups", Method: "POST", URI: "/"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/"},
	{Name: "ListTLSInspectionConfigurations", Method: "POST", URI: "/"},
	{Name: "PutResourcePolicy", Method: "POST", URI: "/"},
	{Name: "TagResource", Method: "POST", URI: "/"},
	{Name: "UntagResource", Method: "POST", URI: "/"},
	{Name: "UpdateFirewallDeleteProtection", Method: "POST", URI: "/"},
	{Name: "UpdateFirewallDescription", Method: "POST", URI: "/"},
	{Name: "UpdateFirewallEncryptionConfiguration", Method: "POST", URI: "/"},
	{Name: "UpdateFirewallPolicy", Method: "POST", URI: "/"},
	{Name: "UpdateFirewallPolicyChangeProtection", Method: "POST", URI: "/"},
	{Name: "UpdateLoggingConfiguration", Method: "POST", URI: "/"},
	{Name: "UpdateRuleGroup", Method: "POST", URI: "/"},
	{Name: "UpdateSubnetChangeProtection", Method: "POST", URI: "/"},
	{Name: "UpdateTLSInspectionConfiguration", Method: "POST", URI: "/"},
}

var networkFirewallOperationByName = func() map[string]networkFirewallOperation {
	out := make(map[string]networkFirewallOperation, len(networkFirewallOperations))
	for _, op := range networkFirewallOperations {
		out[op.Name] = op
	}
	return out
}()
