package server

type elasticLoadBalancingV2Operation struct {
	Name string
}

// AWS Elastic Load Balancing (ELB, 2012-06-01) operations sourced from:
// https://docs.aws.amazon.com/elasticloadbalancing/2012-06-01/APIReference/API_Operations.html
var elasticLoadBalancingV2Operations = []elasticLoadBalancingV2Operation{
	{Name: "AddTags"},
	{Name: "ApplySecurityGroupsToLoadBalancer"},
	{Name: "AttachLoadBalancerToSubnets"},
	{Name: "ConfigureHealthCheck"},
	{Name: "CreateAppCookieStickinessPolicy"},
	{Name: "CreateLBCookieStickinessPolicy"},
	{Name: "CreateLoadBalancer"},
	{Name: "CreateLoadBalancerListeners"},
	{Name: "CreateLoadBalancerPolicy"},
	{Name: "DeleteLoadBalancer"},
	{Name: "DeleteLoadBalancerListeners"},
	{Name: "DeleteLoadBalancerPolicy"},
	{Name: "DeregisterInstancesFromLoadBalancer"},
	{Name: "DescribeAccountLimits"},
	{Name: "DescribeInstanceHealth"},
	{Name: "DescribeLoadBalancerAttributes"},
	{Name: "DescribeLoadBalancerPolicies"},
	{Name: "DescribeLoadBalancerPolicyTypes"},
	{Name: "DescribeLoadBalancers"},
	{Name: "DescribeTags"},
	{Name: "DetachLoadBalancerFromSubnets"},
	{Name: "DisableAvailabilityZonesForLoadBalancer"},
	{Name: "EnableAvailabilityZonesForLoadBalancer"},
	{Name: "ModifyLoadBalancerAttributes"},
	{Name: "RegisterInstancesWithLoadBalancer"},
	{Name: "RemoveTags"},
	{Name: "SetLoadBalancerListenerSSLCertificate"},
	{Name: "SetLoadBalancerPoliciesForBackendServer"},
	{Name: "SetLoadBalancerPoliciesOfListener"},
}

var elasticLoadBalancingV2OperationByName = func() map[string]elasticLoadBalancingV2Operation {
	out := make(map[string]elasticLoadBalancingV2Operation, len(elasticLoadBalancingV2Operations))
	for _, op := range elasticLoadBalancingV2Operations {
		out[op.Name] = op
	}
	return out
}()
