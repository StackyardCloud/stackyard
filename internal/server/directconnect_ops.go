package server

type directConnectOperation struct {
	Name string
}

// AWS Direct Connect operations sourced from:
// https://docs.aws.amazon.com/directconnect/latest/APIReference/API_Operations.html
var directConnectOperations = []directConnectOperation{
	{Name: "AcceptDirectConnectGatewayAssociationProposal"},
	{Name: "AllocateConnectionOnInterconnect"},
	{Name: "AllocateHostedConnection"},
	{Name: "AllocatePrivateVirtualInterface"},
	{Name: "AllocatePublicVirtualInterface"},
	{Name: "AllocateTransitVirtualInterface"},
	{Name: "AssociateConnectionWithLag"},
	{Name: "AssociateHostedConnection"},
	{Name: "AssociateMacSecKey"},
	{Name: "AssociateVirtualInterface"},
	{Name: "ConfirmConnection"},
	{Name: "ConfirmCustomerAgreement"},
	{Name: "ConfirmPrivateVirtualInterface"},
	{Name: "ConfirmPublicVirtualInterface"},
	{Name: "ConfirmTransitVirtualInterface"},
	{Name: "CreateBGPPeer"},
	{Name: "CreateConnection"},
	{Name: "CreateDirectConnectGateway"},
	{Name: "CreateDirectConnectGatewayAssociation"},
	{Name: "CreateDirectConnectGatewayAssociationProposal"},
	{Name: "CreateInterconnect"},
	{Name: "CreateLag"},
	{Name: "CreatePrivateVirtualInterface"},
	{Name: "CreatePublicVirtualInterface"},
	{Name: "CreateTransitVirtualInterface"},
	{Name: "DeleteBGPPeer"},
	{Name: "DeleteConnection"},
	{Name: "DeleteDirectConnectGateway"},
	{Name: "DeleteDirectConnectGatewayAssociation"},
	{Name: "DeleteDirectConnectGatewayAssociationProposal"},
	{Name: "DeleteInterconnect"},
	{Name: "DeleteLag"},
	{Name: "DeleteVirtualInterface"},
	{Name: "DescribeConnectionLoa"},
	{Name: "DescribeConnections"},
	{Name: "DescribeConnectionsOnInterconnect"},
	{Name: "DescribeCustomerMetadata"},
	{Name: "DescribeDirectConnectGatewayAssociationProposals"},
	{Name: "DescribeDirectConnectGatewayAssociations"},
	{Name: "DescribeDirectConnectGatewayAttachments"},
	{Name: "DescribeDirectConnectGateways"},
	{Name: "DescribeHostedConnections"},
	{Name: "DescribeInterconnectLoa"},
	{Name: "DescribeInterconnects"},
	{Name: "DescribeLags"},
	{Name: "DescribeLoa"},
	{Name: "DescribeLocations"},
	{Name: "DescribeRouterConfiguration"},
	{Name: "DescribeTags"},
	{Name: "DescribeVirtualGateways"},
	{Name: "DescribeVirtualInterfaces"},
	{Name: "DisassociateConnectionFromLag"},
	{Name: "DisassociateMacSecKey"},
	{Name: "ListVirtualInterfaceTestHistory"},
	{Name: "StartBgpFailoverTest"},
	{Name: "StopBgpFailoverTest"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateConnection"},
	{Name: "UpdateDirectConnectGateway"},
	{Name: "UpdateDirectConnectGatewayAssociation"},
	{Name: "UpdateLag"},
	{Name: "UpdateVirtualInterfaceAttributes"},
}

var directConnectOperationByName = func() map[string]directConnectOperation {
	out := make(map[string]directConnectOperation, len(directConnectOperations))
	for _, op := range directConnectOperations {
		out[op.Name] = op
	}
	return out
}()
