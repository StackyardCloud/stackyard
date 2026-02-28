package server

type rtbFabricOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS RTB Fabric operations sourced from:
// https://docs.aws.amazon.com/rtb-fabric/latest/APIReference/API_Operations.html
// (The legacy APIReference path redirects to /latest/api/.)
var rtbFabricOperations = []rtbFabricOperation{
	{Name: "AcceptLink", Method: "POST", URI: "/gateway/{gatewayId}/link/{linkId}/accept"},
	{Name: "CreateInboundExternalLink", Method: "POST", URI: "/responder-gateway/{gatewayId}/inbound-external-link"},
	{Name: "CreateLink", Method: "POST", URI: "/gateway/{gatewayId}/create-link"},
	{Name: "CreateOutboundExternalLink", Method: "POST", URI: "/requester-gateway/{gatewayId}/outbound-external-link"},
	{Name: "CreateRequesterGateway", Method: "POST", URI: "/requester-gateway"},
	{Name: "CreateResponderGateway", Method: "POST", URI: "/responder-gateway"},
	{Name: "DeleteInboundExternalLink", Method: "DELETE", URI: "/responder-gateway/{gatewayId}/inbound-external-link/{linkId}"},
	{Name: "DeleteLink", Method: "DELETE", URI: "/gateway/{gatewayId}/link/{linkId}"},
	{Name: "DeleteOutboundExternalLink", Method: "DELETE", URI: "/requester-gateway/{gatewayId}/outbound-external-link/{linkId}"},
	{Name: "DeleteRequesterGateway", Method: "DELETE", URI: "/requester-gateway/{gatewayId}"},
	{Name: "DeleteResponderGateway", Method: "DELETE", URI: "/responder-gateway/{gatewayId}"},
	{Name: "GetInboundExternalLink", Method: "GET", URI: "/responder-gateway/{gatewayId}/inbound-external-link/{linkId}"},
	{Name: "GetLink", Method: "GET", URI: "/gateway/{gatewayId}/link/{linkId}"},
	{Name: "GetOutboundExternalLink", Method: "GET", URI: "/requester-gateway/{gatewayId}/outbound-external-link/{linkId}"},
	{Name: "GetRequesterGateway", Method: "GET", URI: "/requester-gateway/{gatewayId}"},
	{Name: "GetResponderGateway", Method: "GET", URI: "/responder-gateway/{gatewayId}"},
	{Name: "ListLinks", Method: "GET", URI: "/gateway/{gatewayId}/links/?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListRequesterGateways", Method: "GET", URI: "/requester-gateways?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListResponderGateways", Method: "GET", URI: "/responder-gateways?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "RejectLink", Method: "POST", URI: "/gateway/{gatewayId}/link/{linkId}/reject"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateLink", Method: "PATCH", URI: "/gateway/{gatewayId}/link/{linkId}"},
	{Name: "UpdateLinkModuleFlow", Method: "POST", URI: "/gateway/{gatewayId}/link/{linkId}/module-flow"},
	{Name: "UpdateRequesterGateway", Method: "POST", URI: "/requester-gateway/{gatewayId}/update"},
	{Name: "UpdateResponderGateway", Method: "POST", URI: "/responder-gateway/{gatewayId}/update"},
}

var rtbFabricOperationByName = func() map[string]rtbFabricOperation {
	out := make(map[string]rtbFabricOperation, len(rtbFabricOperations))
	for _, op := range rtbFabricOperations {
		out[op.Name] = op
	}
	return out
}()
