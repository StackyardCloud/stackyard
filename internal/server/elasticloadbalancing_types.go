package server

type elasticLoadBalancingDataType struct {
	Name string
}

// AWS Elastic Load Balancing (ELB) data types sourced from:
// https://docs.aws.amazon.com/elasticloadbalancing/latest/APIReference/API_Types.html
var elasticLoadBalancingDataTypes = []elasticLoadBalancingDataType{
	{Name: "Action"},
	{Name: "AdministrativeOverride"},
	{Name: "AnomalyDetection"},
	{Name: "AuthenticateCognitoActionConfig"},
	{Name: "AuthenticateOidcActionConfig"},
	{Name: "AvailabilityZone"},
	{Name: "CapacityReservationStatus"},
	{Name: "Certificate"},
	{Name: "Cipher"},
	{Name: "DescribeTrustStoreRevocation"},
	{Name: "FixedResponseActionConfig"},
	{Name: "ForwardActionConfig"},
	{Name: "HostHeaderConditionConfig"},
	{Name: "HostHeaderRewriteConfig"},
	{Name: "HttpHeaderConditionConfig"},
	{Name: "HttpRequestMethodConditionConfig"},
	{Name: "IpamPools"},
	{Name: "JwtValidationActionAdditionalClaim"},
	{Name: "JwtValidationActionConfig"},
	{Name: "Limit"},
	{Name: "Listener"},
	{Name: "ListenerAttribute"},
	{Name: "LoadBalancer"},
	{Name: "LoadBalancerAddress"},
	{Name: "LoadBalancerAttribute"},
	{Name: "LoadBalancerState"},
	{Name: "Matcher"},
	{Name: "MinimumLoadBalancerCapacity"},
	{Name: "MutualAuthenticationAttributes"},
	{Name: "PathPatternConditionConfig"},
	{Name: "QueryStringConditionConfig"},
	{Name: "QueryStringKeyValuePair"},
	{Name: "RedirectActionConfig"},
	{Name: "RevocationContent"},
	{Name: "RewriteConfig"},
	{Name: "Rule"},
	{Name: "RuleCondition"},
	{Name: "RulePriorityPair"},
	{Name: "RuleTransform"},
	{Name: "SourceIpConditionConfig"},
	{Name: "SslPolicy"},
	{Name: "SubnetMapping"},
	{Name: "Tag"},
	{Name: "TagDescription"},
	{Name: "TargetDescription"},
	{Name: "TargetGroup"},
	{Name: "TargetGroupAttribute"},
	{Name: "TargetGroupStickinessConfig"},
	{Name: "TargetGroupTuple"},
	{Name: "TargetHealth"},
	{Name: "TargetHealthDescription"},
	{Name: "TrustStore"},
	{Name: "TrustStoreAssociation"},
	{Name: "TrustStoreRevocation"},
	{Name: "UrlRewriteConfig"},
	{Name: "ZonalCapacityReservationState"},
	{Name: "SetSubnets"},
}

var elasticLoadBalancingDataTypeByName = func() map[string]elasticLoadBalancingDataType {
	out := make(map[string]elasticLoadBalancingDataType, len(elasticLoadBalancingDataTypes))
	for _, dt := range elasticLoadBalancingDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
