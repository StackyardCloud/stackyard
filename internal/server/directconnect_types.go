package server

type directConnectDataType struct {
	Name string
}

// AWS Direct Connect data types sourced from:
// https://docs.aws.amazon.com/directconnect/latest/APIReference/API_Types.html
var directConnectDataTypes = []directConnectDataType{
	{Name: "AssociatedCoreNetwork"},
	{Name: "AssociatedGateway"},
	{Name: "BGPPeer"},
	{Name: "Connection"},
	{Name: "CustomerAgreement"},
	{Name: "DirectConnectGateway"},
	{Name: "DirectConnectGatewayAssociation"},
	{Name: "DirectConnectGatewayAssociationProposal"},
	{Name: "DirectConnectGatewayAttachment"},
	{Name: "Interconnect"},
	{Name: "Lag"},
	{Name: "Loa"},
	{Name: "Location"},
	{Name: "MacSecKey"},
	{Name: "NewBGPPeer"},
	{Name: "NewPrivateVirtualInterface"},
	{Name: "NewPrivateVirtualInterfaceAllocation"},
	{Name: "NewPublicVirtualInterface"},
	{Name: "NewPublicVirtualInterfaceAllocation"},
	{Name: "NewTransitVirtualInterface"},
	{Name: "NewTransitVirtualInterfaceAllocation"},
	{Name: "ResourceTag"},
	{Name: "RouteFilterPrefix"},
	{Name: "RouterType"},
	{Name: "Tag"},
	{Name: "VirtualGateway"},
	{Name: "VirtualInterface"},
	{Name: "VirtualInterfaceTestHistory"},
	{Name: "UpdateVirtualInterfaceAttributes"},
}

var directConnectDataTypeByName = func() map[string]directConnectDataType {
	out := make(map[string]directConnectDataType, len(directConnectDataTypes))
	for _, t := range directConnectDataTypes {
		out[t.Name] = t
	}
	return out
}()
