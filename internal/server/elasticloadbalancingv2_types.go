package server

type elasticLoadBalancingV2DataType struct {
	Name string
}

// AWS Elastic Load Balancing (ELB, 2012-06-01) data types sourced from:
// https://docs.aws.amazon.com/elasticloadbalancing/2012-06-01/APIReference/API_Types.html
var elasticLoadBalancingV2DataTypes = []elasticLoadBalancingV2DataType{
	{Name: "AccessLog"},
	{Name: "AdditionalAttribute"},
	{Name: "AppCookieStickinessPolicy"},
	{Name: "BackendServerDescription"},
	{Name: "ConnectionDraining"},
	{Name: "ConnectionSettings"},
	{Name: "CrossZoneLoadBalancing"},
	{Name: "HealthCheck"},
	{Name: "Instance"},
	{Name: "InstanceState"},
	{Name: "LBCookieStickinessPolicy"},
	{Name: "Limit"},
	{Name: "Listener"},
	{Name: "ListenerDescription"},
	{Name: "LoadBalancerAttributes"},
	{Name: "LoadBalancerDescription"},
	{Name: "Policies"},
	{Name: "PolicyAttribute"},
	{Name: "PolicyAttributeDescription"},
	{Name: "PolicyAttributeTypeDescription"},
	{Name: "PolicyDescription"},
	{Name: "PolicyTypeDescription"},
	{Name: "SourceSecurityGroup"},
	{Name: "Tag"},
	{Name: "TagDescription"},
	{Name: "TagKeyOnly"},
}

var elasticLoadBalancingV2DataTypeByName = func() map[string]elasticLoadBalancingV2DataType {
	out := make(map[string]elasticLoadBalancingV2DataType, len(elasticLoadBalancingV2DataTypes))
	for _, dt := range elasticLoadBalancingV2DataTypes {
		out[dt.Name] = dt
	}
	return out
}()
