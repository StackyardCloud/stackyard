package server

type networkFirewallDataType struct {
	Name string
}

// AWS Network Firewall data types sourced from:
// https://docs.aws.amazon.com/network-firewall/latest/APIReference/API_Types.html
var networkFirewallDataTypes = []networkFirewallDataType{
	{Name: "ActionDefinition"},
	{Name: "Address"},
	{Name: "AnalysisResult"},
	{Name: "Attachment"},
	{Name: "CIDRSummary"},
	{Name: "CapacityUsageSummary"},
	{Name: "CheckCertificateRevocationStatusActions"},
	{Name: "CustomAction"},
	{Name: "Dimension"},
	{Name: "EncryptionConfiguration"},
	{Name: "Firewall"},
	{Name: "FirewallMetadata"},
	{Name: "FirewallPolicy"},
	{Name: "FirewallPolicyMetadata"},
	{Name: "FirewallStatus"},
	{Name: "FlowTimeouts"},
	{Name: "Header"},
	{Name: "IPSet"},
	{Name: "IPSetMetadata"},
	{Name: "IPSetReference"},
	{Name: "LogDestinationConfig"},
	{Name: "LoggingConfiguration"},
	{Name: "MatchAttributes"},
	{Name: "PerObjectStatus"},
	{Name: "PolicyVariables"},
	{Name: "PortRange"},
	{Name: "PortSet"},
	{Name: "PublishMetricAction"},
	{Name: "ReferenceSets"},
	{Name: "RuleDefinition"},
	{Name: "RuleGroup"},
	{Name: "RuleGroupMetadata"},
	{Name: "RuleOption"},
	{Name: "RuleVariables"},
	{Name: "RulesSource"},
	{Name: "RulesSourceList"},
	{Name: "ServerCertificate"},
	{Name: "ServerCertificateConfiguration"},
	{Name: "ServerCertificateScope"},
	{Name: "SourceMetadata"},
	{Name: "StatefulEngineOptions"},
	{Name: "StatefulRule"},
	{Name: "StatefulRuleGroupOverride"},
	{Name: "StatefulRuleGroupReference"},
	{Name: "StatefulRuleOptions"},
	{Name: "StatelessRule"},
	{Name: "StatelessRuleGroupReference"},
	{Name: "StatelessRulesAndCustomActions"},
	{Name: "SubnetMapping"},
	{Name: "SyncState"},
	{Name: "TCPFlagField"},
	{Name: "Tag"},
	{Name: "TLSInspectionConfiguration"},
	{Name: "TLSInspectionConfigurationMetadata"},
	{Name: "TlsCertificateData"},
}

var networkFirewallDataTypeByName = func() map[string]networkFirewallDataType {
	out := make(map[string]networkFirewallDataType, len(networkFirewallDataTypes))
	for _, dt := range networkFirewallDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
