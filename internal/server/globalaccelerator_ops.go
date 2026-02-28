package server

type globalAcceleratorOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Global Accelerator operations sourced from:
// https://docs.aws.amazon.com/global-accelerator/latest/api/API_Operations.html
// (The legacy APIReference path redirects to /latest/api/.)
var globalAcceleratorOperations = []globalAcceleratorOperation{
	{Name: "AddCustomRoutingEndpoints", Method: "POST", URI: "/"},
	{Name: "AddEndpoints", Method: "POST", URI: "/"},
	{Name: "AdvertiseByoipCidr", Method: "POST", URI: "/"},
	{Name: "AllowCustomRoutingTraffic", Method: "POST", URI: "/"},
	{Name: "CreateAccelerator", Method: "POST", URI: "/"},
	{Name: "CreateCrossAccountAttachment", Method: "POST", URI: "/"},
	{Name: "CreateCustomRoutingAccelerator", Method: "POST", URI: "/"},
	{Name: "CreateCustomRoutingEndpointGroup", Method: "POST", URI: "/"},
	{Name: "CreateCustomRoutingListener", Method: "POST", URI: "/"},
	{Name: "CreateEndpointGroup", Method: "POST", URI: "/"},
	{Name: "CreateListener", Method: "POST", URI: "/"},
	{Name: "DeleteAccelerator", Method: "POST", URI: "/"},
	{Name: "DeleteCrossAccountAttachment", Method: "POST", URI: "/"},
	{Name: "DeleteCustomRoutingAccelerator", Method: "POST", URI: "/"},
	{Name: "DeleteCustomRoutingEndpointGroup", Method: "POST", URI: "/"},
	{Name: "DeleteCustomRoutingListener", Method: "POST", URI: "/"},
	{Name: "DeleteEndpointGroup", Method: "POST", URI: "/"},
	{Name: "DeleteListener", Method: "POST", URI: "/"},
	{Name: "DenyCustomRoutingTraffic", Method: "POST", URI: "/"},
	{Name: "DeprovisionByoipCidr", Method: "POST", URI: "/"},
	{Name: "DescribeAccelerator", Method: "POST", URI: "/"},
	{Name: "DescribeAcceleratorAttributes", Method: "POST", URI: "/"},
	{Name: "DescribeCrossAccountAttachment", Method: "POST", URI: "/"},
	{Name: "DescribeCustomRoutingAccelerator", Method: "POST", URI: "/"},
	{Name: "DescribeCustomRoutingAcceleratorAttributes", Method: "POST", URI: "/"},
	{Name: "DescribeCustomRoutingEndpointGroup", Method: "POST", URI: "/"},
	{Name: "DescribeCustomRoutingListener", Method: "POST", URI: "/"},
	{Name: "DescribeEndpointGroup", Method: "POST", URI: "/"},
	{Name: "DescribeListener", Method: "POST", URI: "/"},
	{Name: "ListAccelerators", Method: "POST", URI: "/"},
	{Name: "ListByoipCidrs", Method: "POST", URI: "/"},
	{Name: "ListCrossAccountAttachments", Method: "POST", URI: "/"},
	{Name: "ListCrossAccountResourceAccounts", Method: "POST", URI: "/"},
	{Name: "ListCrossAccountResources", Method: "POST", URI: "/"},
	{Name: "ListCustomRoutingAccelerators", Method: "POST", URI: "/"},
	{Name: "ListCustomRoutingEndpointGroups", Method: "POST", URI: "/"},
	{Name: "ListCustomRoutingListeners", Method: "POST", URI: "/"},
	{Name: "ListCustomRoutingPortMappings", Method: "POST", URI: "/"},
	{Name: "ListCustomRoutingPortMappingsByDestination", Method: "POST", URI: "/"},
	{Name: "ListEndpointGroups", Method: "POST", URI: "/"},
	{Name: "ListListeners", Method: "POST", URI: "/"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/"},
	{Name: "ProvisionByoipCidr", Method: "POST", URI: "/"},
	{Name: "RemoveCustomRoutingEndpoints", Method: "POST", URI: "/"},
	{Name: "RemoveEndpoints", Method: "POST", URI: "/"},
	{Name: "TagResource", Method: "POST", URI: "/"},
	{Name: "UntagResource", Method: "POST", URI: "/"},
	{Name: "UpdateAccelerator", Method: "POST", URI: "/"},
	{Name: "UpdateAcceleratorAttributes", Method: "POST", URI: "/"},
	{Name: "UpdateCrossAccountAttachment", Method: "POST", URI: "/"},
	{Name: "UpdateCustomRoutingAccelerator", Method: "POST", URI: "/"},
	{Name: "UpdateCustomRoutingAcceleratorAttributes", Method: "POST", URI: "/"},
	{Name: "UpdateCustomRoutingListener", Method: "POST", URI: "/"},
	{Name: "UpdateEndpointGroup", Method: "POST", URI: "/"},
	{Name: "UpdateListener", Method: "POST", URI: "/"},
	{Name: "WithdrawByoipCidr", Method: "POST", URI: "/"},
}

var globalAcceleratorOperationByName = func() map[string]globalAcceleratorOperation {
	out := make(map[string]globalAcceleratorOperation, len(globalAcceleratorOperations))
	for _, op := range globalAcceleratorOperations {
		out[op.Name] = op
	}
	return out
}()
