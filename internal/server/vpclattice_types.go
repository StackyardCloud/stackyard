package server

type vpcLatticeDataType struct {
	Name string
}

// AWS VPC Lattice data types sourced from:
// https://docs.aws.amazon.com/vpc-lattice/latest/APIReference/API_Types.html
var vpcLatticeDataTypes = []vpcLatticeDataType{
	{Name: "AccessLogSubscriptionSummary"},
	{Name: "ArnResource"},
	{Name: "DnsEntry"},
	{Name: "DnsOptions"},
	{Name: "DnsResource"},
	{Name: "DomainVerificationSummary"},
	{Name: "FixedResponseAction"},
	{Name: "ForwardAction"},
	{Name: "HeaderMatch"},
	{Name: "HeaderMatchType"},
	{Name: "HealthCheckConfig"},
	{Name: "HttpMatch"},
	{Name: "IpResource"},
	{Name: "ListenerSummary"},
	{Name: "Matcher"},
	{Name: "PathMatch"},
	{Name: "PathMatchType"},
	{Name: "ResourceConfigurationDefinition"},
	{Name: "ResourceConfigurationSummary"},
	{Name: "ResourceEndpointAssociationSummary"},
	{Name: "ResourceGatewaySummary"},
	{Name: "RuleAction"},
	{Name: "RuleMatch"},
	{Name: "RuleSummary"},
	{Name: "RuleUpdate"},
	{Name: "RuleUpdateFailure"},
	{Name: "RuleUpdateSuccess"},
	{Name: "ServiceNetworkEndpointAssociation"},
	{Name: "ServiceNetworkResourceAssociationSummary"},
	{Name: "ServiceNetworkServiceAssociationSummary"},
	{Name: "ServiceNetworkSummary"},
	{Name: "ServiceNetworkVpcAssociationSummary"},
	{Name: "ServiceSummary"},
	{Name: "SharingConfig"},
	{Name: "Target"},
	{Name: "TargetFailure"},
	{Name: "TargetGroupConfig"},
	{Name: "TargetGroupSummary"},
	{Name: "TargetSummary"},
	{Name: "TxtMethodConfig"},
	{Name: "ValidationExceptionField"},
	{Name: "WeightedTargetGroup"},
}

var vpcLatticeDataTypeByName = func() map[string]vpcLatticeDataType {
	out := make(map[string]vpcLatticeDataType, len(vpcLatticeDataTypes))
	for _, dt := range vpcLatticeDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
