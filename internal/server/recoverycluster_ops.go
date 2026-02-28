package server

type recoveryClusterOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Route 53 Application Recovery Controller - Recovery Control Configuration operations sourced from:
// https://docs.aws.amazon.com/recovery-cluster/latest/APIReference/API_Operations.html
// AWS currently serves this API guide under /latest/api/ resource pages.
var recoveryClusterOperations = []recoveryClusterOperation{
	{Name: "CreateCluster", Method: "POST", URI: "/cluster"},
	{Name: "CreateControlPanel", Method: "POST", URI: "/controlpanel"},
	{Name: "CreateRoutingControl", Method: "POST", URI: "/routingcontrol"},
	{Name: "CreateSafetyRule", Method: "POST", URI: "/safetyrule"},
	{Name: "DeleteCluster", Method: "DELETE", URI: "/cluster/{ClusterArn}"},
	{Name: "DeleteControlPanel", Method: "DELETE", URI: "/controlpanel/{ControlPanelArn}"},
	{Name: "DeleteRoutingControl", Method: "DELETE", URI: "/routingcontrol/{RoutingControlArn}"},
	{Name: "DeleteSafetyRule", Method: "DELETE", URI: "/safetyrule/{SafetyRuleArn}"},
	{Name: "DescribeCluster", Method: "GET", URI: "/cluster/{ClusterArn}"},
	{Name: "DescribeControlPanel", Method: "GET", URI: "/controlpanel/{ControlPanelArn}"},
	{Name: "DescribeRoutingControl", Method: "GET", URI: "/routingcontrol/{RoutingControlArn}"},
	{Name: "DescribeSafetyRule", Method: "GET", URI: "/safetyrule/{SafetyRuleArn}"},
	{Name: "GetResourcePolicy", Method: "GET", URI: "/resourcePolicy/{ResourceArn}"},
	{Name: "ListAssociatedRoute53HealthChecks", Method: "GET", URI: "/routingcontrol/{RoutingControlArn}/associatedRoute53HealthChecks"},
	{Name: "ListClusters", Method: "GET", URI: "/cluster"},
	{Name: "ListControlPanels", Method: "GET", URI: "/controlpanels"},
	{Name: "ListRoutingControls", Method: "GET", URI: "/controlpanel/{ControlPanelArn}/routingcontrols"},
	{Name: "ListSafetyRules", Method: "GET", URI: "/controlpanel/{ControlPanelArn}/safetyrules"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateCluster", Method: "PUT", URI: "/cluster"},
	{Name: "UpdateControlPanel", Method: "PUT", URI: "/controlpanel"},
	{Name: "UpdateRoutingControl", Method: "PUT", URI: "/routingcontrol"},
	{Name: "UpdateSafetyRule", Method: "PUT", URI: "/safetyrule"},
}

var recoveryClusterOperationByName = func() map[string]recoveryClusterOperation {
	out := make(map[string]recoveryClusterOperation, len(recoveryClusterOperations))
	for _, op := range recoveryClusterOperations {
		out[op.Name] = op
	}
	return out
}()
