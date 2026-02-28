package server

type mediaConnectOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Elemental MediaConnect operations sourced from:
// https://docs.aws.amazon.com/mediaconnect/latest/api/API_Operations.html
var mediaConnectOperations = []mediaConnectOperation{
	{Name: "AddBridgeOutputs", Method: "POST", URI: "/v1/bridges/{BridgeArn}/outputs"},
	{Name: "AddBridgeSources", Method: "POST", URI: "/v1/bridges/{BridgeArn}/sources"},
	{Name: "AddFlowMediaStreams", Method: "POST", URI: "/v1/flows/{FlowArn}/mediaStreams"},
	{Name: "AddFlowOutputs", Method: "POST", URI: "/v1/flows/{FlowArn}/outputs"},
	{Name: "AddFlowSources", Method: "POST", URI: "/v1/flows/{FlowArn}/source"},
	{Name: "AddFlowVpcInterfaces", Method: "POST", URI: "/v1/flows/{FlowArn}/vpcInterfaces"},
	{Name: "BatchGetRouterInput", Method: "GET", URI: "/v1/routerInputs"},
	{Name: "BatchGetRouterNetworkInterface", Method: "GET", URI: "/v1/routerNetworkInterfaces"},
	{Name: "BatchGetRouterOutput", Method: "GET", URI: "/v1/routerOutputs"},
	{Name: "CreateBridge", Method: "POST", URI: "/v1/bridges"},
	{Name: "CreateFlow", Method: "POST", URI: "/v1/flows"},
	{Name: "CreateGateway", Method: "POST", URI: "/v1/gateways"},
	{Name: "CreateRouterInput", Method: "POST", URI: "/v1/routerInput"},
	{Name: "CreateRouterNetworkInterface", Method: "POST", URI: "/v1/routerNetworkInterface"},
	{Name: "CreateRouterOutput", Method: "POST", URI: "/v1/routerOutput"},
	{Name: "DeleteBridge", Method: "DELETE", URI: "/v1/bridges/{BridgeArn}"},
	{Name: "DeleteFlow", Method: "DELETE", URI: "/v1/flows/{FlowArn}"},
	{Name: "DeleteGateway", Method: "DELETE", URI: "/v1/gateways/{GatewayArn}"},
	{Name: "DeleteRouterInput", Method: "DELETE", URI: "/v1/routerInput/{Arn}"},
	{Name: "DeleteRouterNetworkInterface", Method: "DELETE", URI: "/v1/routerNetworkInterface/{Arn}"},
	{Name: "DeleteRouterOutput", Method: "DELETE", URI: "/v1/routerOutput/{Arn}"},
	{Name: "DeregisterGatewayInstance", Method: "DELETE", URI: "/v1/gateway-instances/{GatewayInstanceArn}"},
	{Name: "DescribeBridge", Method: "GET", URI: "/v1/bridges/{BridgeArn}"},
	{Name: "DescribeFlow", Method: "GET", URI: "/v1/flows/{FlowArn}"},
	{Name: "DescribeFlowSourceMetadata", Method: "GET", URI: "/v1/flows/{FlowArn}/source-metadata"},
	{Name: "DescribeFlowSourceThumbnail", Method: "GET", URI: "/v1/flows/{FlowArn}/source-thumbnail"},
	{Name: "DescribeGateway", Method: "GET", URI: "/v1/gateways/{GatewayArn}"},
	{Name: "DescribeGatewayInstance", Method: "GET", URI: "/v1/gateway-instances/{GatewayInstanceArn}"},
	{Name: "DescribeOffering", Method: "GET", URI: "/v1/offerings/{OfferingArn}"},
	{Name: "DescribeReservation", Method: "GET", URI: "/v1/reservations/{ReservationArn}"},
	{Name: "GetRouterInput", Method: "GET", URI: "/v1/routerInput/{Arn}"},
	{Name: "GetRouterInputSourceMetadata", Method: "GET", URI: "/v1/routerInput/{Arn}/source-metadata"},
	{Name: "GetRouterInputThumbnail", Method: "GET", URI: "/v1/routerInput/{Arn}/thumbnail"},
	{Name: "GetRouterNetworkInterface", Method: "GET", URI: "/v1/routerNetworkInterface/{Arn}"},
	{Name: "GetRouterOutput", Method: "GET", URI: "/v1/routerOutput/{Arn}"},
	{Name: "GrantFlowEntitlements", Method: "POST", URI: "/v1/flows/{FlowArn}/entitlements"},
	{Name: "ListBridges", Method: "GET", URI: "/v1/bridges"},
	{Name: "ListEntitlements", Method: "GET", URI: "/v1/entitlements"},
	{Name: "ListFlows", Method: "GET", URI: "/v1/flows"},
	{Name: "ListGatewayInstances", Method: "GET", URI: "/v1/gateway-instances"},
	{Name: "ListGateways", Method: "GET", URI: "/v1/gateways"},
	{Name: "ListOfferings", Method: "GET", URI: "/v1/offerings"},
	{Name: "ListReservations", Method: "GET", URI: "/v1/reservations"},
	{Name: "ListRouterInputs", Method: "POST", URI: "/v1/routerInputs"},
	{Name: "ListRouterNetworkInterfaces", Method: "POST", URI: "/v1/routerNetworkInterfaces"},
	{Name: "ListRouterOutputs", Method: "POST", URI: "/v1/routerOutputs"},
	{Name: "ListTagsForGlobalResource", Method: "GET", URI: "/tags/global/{ResourceArn}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "PurchaseOffering", Method: "POST", URI: "/v1/offerings/{OfferingArn}"},
	{Name: "RemoveBridgeOutput", Method: "DELETE", URI: "/v1/bridges/{BridgeArn}/outputs/{OutputName}"},
	{Name: "RemoveBridgeSource", Method: "DELETE", URI: "/v1/bridges/{BridgeArn}/sources/{SourceName}"},
	{Name: "RemoveFlowMediaStream", Method: "DELETE", URI: "/v1/flows/{FlowArn}/mediaStreams/{MediaStreamName}"},
	{Name: "RemoveFlowOutput", Method: "DELETE", URI: "/v1/flows/{FlowArn}/outputs/{OutputArn}"},
	{Name: "RemoveFlowSource", Method: "DELETE", URI: "/v1/flows/{FlowArn}/source/{SourceArn}"},
	{Name: "RemoveFlowVpcInterface", Method: "DELETE", URI: "/v1/flows/{FlowArn}/vpcInterfaces/{VpcInterfaceName}"},
	{Name: "RestartRouterInput", Method: "POST", URI: "/v1/routerInput/restart/{Arn}"},
	{Name: "RestartRouterOutput", Method: "POST", URI: "/v1/routerOutput/restart/{Arn}"},
	{Name: "RevokeFlowEntitlement", Method: "DELETE", URI: "/v1/flows/{FlowArn}/entitlements/{EntitlementArn}"},
	{Name: "StartFlow", Method: "POST", URI: "/v1/flows/start/{FlowArn}"},
	{Name: "StartRouterInput", Method: "POST", URI: "/v1/routerInput/start/{Arn}"},
	{Name: "StartRouterOutput", Method: "POST", URI: "/v1/routerOutput/start/{Arn}"},
	{Name: "StopFlow", Method: "POST", URI: "/v1/flows/stop/{FlowArn}"},
	{Name: "StopRouterInput", Method: "POST", URI: "/v1/routerInput/stop/{Arn}"},
	{Name: "StopRouterOutput", Method: "POST", URI: "/v1/routerOutput/stop/{Arn}"},
	{Name: "TagGlobalResource", Method: "POST", URI: "/tags/global/{ResourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "TakeRouterInput", Method: "PUT", URI: "/v1/routerOutput/takeRouterInput/{RouterOutputArn}"},
	{Name: "UntagGlobalResource", Method: "DELETE", URI: "/tags/global/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateBridge", Method: "PUT", URI: "/v1/bridges/{BridgeArn}"},
	{Name: "UpdateBridgeOutput", Method: "PUT", URI: "/v1/bridges/{BridgeArn}/outputs/{OutputName}"},
	{Name: "UpdateBridgeSource", Method: "PUT", URI: "/v1/bridges/{BridgeArn}/sources/{SourceName}"},
	{Name: "UpdateBridgeState", Method: "PUT", URI: "/v1/bridges/{BridgeArn}/state"},
	{Name: "UpdateFlow", Method: "PUT", URI: "/v1/flows/{FlowArn}"},
	{Name: "UpdateFlowEntitlement", Method: "PUT", URI: "/v1/flows/{FlowArn}/entitlements/{EntitlementArn}"},
	{Name: "UpdateFlowMediaStream", Method: "PUT", URI: "/v1/flows/{FlowArn}/mediaStreams/{MediaStreamName}"},
	{Name: "UpdateFlowOutput", Method: "PUT", URI: "/v1/flows/{FlowArn}/outputs/{OutputArn}"},
	{Name: "UpdateFlowSource", Method: "PUT", URI: "/v1/flows/{FlowArn}/source/{SourceArn}"},
	{Name: "UpdateGatewayInstance", Method: "PUT", URI: "/v1/gateway-instances/{GatewayInstanceArn}"},
	{Name: "UpdateRouterInput", Method: "PUT", URI: "/v1/routerInput/{Arn}"},
	{Name: "UpdateRouterNetworkInterface", Method: "PUT", URI: "/v1/routerNetworkInterface/{Arn}"},
	{Name: "UpdateRouterOutput", Method: "PUT", URI: "/v1/routerOutput/{Arn}"},
}

var mediaConnectOperationByName = func() map[string]mediaConnectOperation {
	out := make(map[string]mediaConnectOperation, len(mediaConnectOperations))
	for _, op := range mediaConnectOperations {
		out[op.Name] = op
	}
	return out
}()
