package server

type globalAcceleratorDataType struct {
	Name string
}

// AWS Global Accelerator data types sourced from:
// https://docs.aws.amazon.com/global-accelerator/latest/api/API_Types.html
// (The legacy APIReference path redirects to /latest/api/.)
var globalAcceleratorDataTypes = []globalAcceleratorDataType{
	{Name: "Accelerator"},
	{Name: "AcceleratorAttributes"},
	{Name: "AcceleratorEvent"},
	{Name: "Attachment"},
	{Name: "ByoipCidr"},
	{Name: "ByoipCidrEvent"},
	{Name: "CidrAuthorizationContext"},
	{Name: "CrossAccountResource"},
	{Name: "CustomRoutingAccelerator"},
	{Name: "CustomRoutingAcceleratorAttributes"},
	{Name: "CustomRoutingDestinationConfiguration"},
	{Name: "CustomRoutingDestinationDescription"},
	{Name: "CustomRoutingEndpointConfiguration"},
	{Name: "CustomRoutingEndpointDescription"},
	{Name: "CustomRoutingEndpointGroup"},
	{Name: "CustomRoutingListener"},
	{Name: "DestinationPortMapping"},
	{Name: "EndpointConfiguration"},
	{Name: "EndpointDescription"},
	{Name: "EndpointGroup"},
	{Name: "EndpointIdentifier"},
	{Name: "IpSet"},
	{Name: "Listener"},
	{Name: "PortMapping"},
	{Name: "PortOverride"},
	{Name: "PortRange"},
	{Name: "Resource"},
	{Name: "SocketAddress"},
	{Name: "Tag"},
}

var globalAcceleratorDataTypeByName = func() map[string]globalAcceleratorDataType {
	out := make(map[string]globalAcceleratorDataType, len(globalAcceleratorDataTypes))
	for _, dt := range globalAcceleratorDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
