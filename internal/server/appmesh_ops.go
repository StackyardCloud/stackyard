package server

type appMeshOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS App Mesh operations sourced from:
// https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_Operations.html
var appMeshOperations = []appMeshOperation{
	{Name: "CreateGatewayRoute", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualGateway/{virtualGatewayName}/gatewayRoutes"},
	{Name: "CreateMesh", Method: "PUT", URI: "/v20190125/meshes"},
	{Name: "CreateRoute", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualRouter/{virtualRouterName}/routes"},
	{Name: "CreateVirtualGateway", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualGateways"},
	{Name: "CreateVirtualNode", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualNodes"},
	{Name: "CreateVirtualRouter", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualRouters"},
	{Name: "CreateVirtualService", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualServices"},
	{Name: "DeleteGatewayRoute", Method: "DELETE", URI: "/v20190125/meshes/{meshName}/virtualGateway/{virtualGatewayName}/gatewayRoutes/{gatewayRouteName}"},
	{Name: "DeleteMesh", Method: "DELETE", URI: "/v20190125/meshes/{meshName}"},
	{Name: "DeleteRoute", Method: "DELETE", URI: "/v20190125/meshes/{meshName}/virtualRouter/{virtualRouterName}/routes/{routeName}"},
	{Name: "DeleteVirtualGateway", Method: "DELETE", URI: "/v20190125/meshes/{meshName}/virtualGateways/{virtualGatewayName}"},
	{Name: "DeleteVirtualNode", Method: "DELETE", URI: "/v20190125/meshes/{meshName}/virtualNodes/{virtualNodeName}"},
	{Name: "DeleteVirtualRouter", Method: "DELETE", URI: "/v20190125/meshes/{meshName}/virtualRouters/{virtualRouterName}"},
	{Name: "DeleteVirtualService", Method: "DELETE", URI: "/v20190125/meshes/{meshName}/virtualServices/{virtualServiceName}"},
	{Name: "DescribeGatewayRoute", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualGateway/{virtualGatewayName}/gatewayRoutes/{gatewayRouteName}"},
	{Name: "DescribeMesh", Method: "GET", URI: "/v20190125/meshes/{meshName}"},
	{Name: "DescribeRoute", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualRouter/{virtualRouterName}/routes/{routeName}"},
	{Name: "DescribeVirtualGateway", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualGateways/{virtualGatewayName}"},
	{Name: "DescribeVirtualNode", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualNodes/{virtualNodeName}"},
	{Name: "DescribeVirtualRouter", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualRouters/{virtualRouterName}"},
	{Name: "DescribeVirtualService", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualServices/{virtualServiceName}"},
	{Name: "ListGatewayRoutes", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualGateway/{virtualGatewayName}/gatewayRoutes"},
	{Name: "ListMeshes", Method: "GET", URI: "/v20190125/meshes"},
	{Name: "ListRoutes", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualRouter/{virtualRouterName}/routes"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/v20190125/tags"},
	{Name: "ListVirtualGateways", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualGateways"},
	{Name: "ListVirtualNodes", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualNodes"},
	{Name: "ListVirtualRouters", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualRouters"},
	{Name: "ListVirtualServices", Method: "GET", URI: "/v20190125/meshes/{meshName}/virtualServices"},
	{Name: "TagResource", Method: "PUT", URI: "/v20190125/tag"},
	{Name: "UntagResource", Method: "PUT", URI: "/v20190125/untag"},
	{Name: "UpdateGatewayRoute", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualGateway/{virtualGatewayName}/gatewayRoutes/{gatewayRouteName}"},
	{Name: "UpdateMesh", Method: "PUT", URI: "/v20190125/meshes/{meshName}"},
	{Name: "UpdateRoute", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualRouter/{virtualRouterName}/routes/{routeName}"},
	{Name: "UpdateVirtualGateway", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualGateways/{virtualGatewayName}"},
	{Name: "UpdateVirtualNode", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualNodes/{virtualNodeName}"},
	{Name: "UpdateVirtualRouter", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualRouters/{virtualRouterName}"},
	{Name: "UpdateVirtualService", Method: "PUT", URI: "/v20190125/meshes/{meshName}/virtualServices/{virtualServiceName}"},
}

var appMeshOperationByName = func() map[string]appMeshOperation {
	out := make(map[string]appMeshOperation, len(appMeshOperations))
	for _, op := range appMeshOperations {
		out[op.Name] = op
	}
	return out
}()
