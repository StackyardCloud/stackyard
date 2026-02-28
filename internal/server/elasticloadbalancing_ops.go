package server

type elasticLoadBalancingOperation struct {
	Name string
}

// AWS Elastic Load Balancing (ELB) operations sourced from:
// https://docs.aws.amazon.com/elasticloadbalancing/latest/APIReference/API_Operations.html
var elasticLoadBalancingOperations = []elasticLoadBalancingOperation{
	{Name: "AddListenerCertificates"},
	{Name: "AddTags"},
	{Name: "AddTrustStoreRevocations"},
	{Name: "CreateListener"},
	{Name: "CreateLoadBalancer"},
	{Name: "CreateRule"},
	{Name: "CreateTargetGroup"},
	{Name: "CreateTrustStore"},
	{Name: "DeleteListener"},
	{Name: "DeleteLoadBalancer"},
	{Name: "DeleteRule"},
	{Name: "DeleteSharedTrustStoreAssociation"},
	{Name: "DeleteTargetGroup"},
	{Name: "DeleteTrustStore"},
	{Name: "DeregisterTargets"},
	{Name: "DescribeAccountLimits"},
	{Name: "DescribeCapacityReservation"},
	{Name: "DescribeListenerAttributes"},
	{Name: "DescribeListenerCertificates"},
	{Name: "DescribeListeners"},
	{Name: "DescribeLoadBalancerAttributes"},
	{Name: "DescribeLoadBalancers"},
	{Name: "DescribeRules"},
	{Name: "DescribeSSLPolicies"},
	{Name: "DescribeTags"},
	{Name: "DescribeTargetGroupAttributes"},
	{Name: "DescribeTargetGroups"},
	{Name: "DescribeTargetHealth"},
	{Name: "DescribeTrustStoreAssociations"},
	{Name: "DescribeTrustStoreRevocations"},
	{Name: "DescribeTrustStores"},
	{Name: "GetResourcePolicy"},
	{Name: "GetTrustStoreCaCertificatesBundle"},
	{Name: "GetTrustStoreRevocationContent"},
	{Name: "ModifyCapacityReservation"},
	{Name: "ModifyIpPools"},
	{Name: "ModifyListener"},
	{Name: "ModifyListenerAttributes"},
	{Name: "ModifyLoadBalancerAttributes"},
	{Name: "ModifyRule"},
	{Name: "ModifyTargetGroup"},
	{Name: "ModifyTargetGroupAttributes"},
	{Name: "ModifyTrustStore"},
	{Name: "RegisterTargets"},
	{Name: "RemoveListenerCertificates"},
	{Name: "RemoveTags"},
	{Name: "RemoveTrustStoreRevocations"},
	{Name: "SetIpAddressType"},
	{Name: "SetRulePriorities"},
	{Name: "SetSecurityGroups"},
	{Name: "SetSubnets"},
}

var elasticLoadBalancingOperationByName = func() map[string]elasticLoadBalancingOperation {
	out := make(map[string]elasticLoadBalancingOperation, len(elasticLoadBalancingOperations))
	for _, op := range elasticLoadBalancingOperations {
		out[op.Name] = op
	}
	return out
}()
