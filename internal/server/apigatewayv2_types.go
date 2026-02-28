package server

type apiGatewayV2DataType struct {
	Name string
}

// AWS API Gateway v2 data types sourced from:
// https://docs.aws.amazon.com/apigatewayv2/latest/api-reference/resources.html
// and https://docs.aws.amazon.com/apigatewayv2/latest/api-reference/toc-contents.json
var apiGatewayV2DataTypes = []apiGatewayV2DataType{
	{Name: "AccessLogSettings"},
	{Name: "Api"},
	{Name: "ApiMapping"},
	{Name: "ApiMappings"},
	{Name: "Apis"},
	{Name: "Authorizer"},
	{Name: "Authorizers"},
	{Name: "AuthorizersCache"},
	{Name: "Cors"},
	{Name: "Deployment"},
	{Name: "Deployments"},
	{Name: "DomainName"},
	{Name: "DomainNames"},
	{Name: "ExportedAPI"},
	{Name: "Integration"},
	{Name: "IntegrationResponse"},
	{Name: "IntegrationResponses"},
	{Name: "Integrations"},
	{Name: "Model"},
	{Name: "Models"},
	{Name: "ModelTemplate"},
	{Name: "Portal"},
	{Name: "Portal product"},
	{Name: "Portal products"},
	{Name: "Portals"},
	{Name: "Preview"},
	{Name: "Product page"},
	{Name: "Product pages"},
	{Name: "Product REST endpoint page"},
	{Name: "Product REST endpoint pages"},
	{Name: "Publish"},
	{Name: "Route"},
	{Name: "RouteRequestParameter"},
	{Name: "RouteResponse"},
	{Name: "RouteResponses"},
	{Name: "Routes"},
	{Name: "RouteSettings"},
	{Name: "RoutingRule"},
	{Name: "RoutingRules"},
	{Name: "Sharing policy"},
	{Name: "Stage"},
	{Name: "Stages"},
	{Name: "Tags"},
	{Name: "VPCLink"},
	{Name: "VPCLinks"},
}

var apiGatewayV2DataTypeByName = func() map[string]apiGatewayV2DataType {
	out := make(map[string]apiGatewayV2DataType, len(apiGatewayV2DataTypes))
	for _, t := range apiGatewayV2DataTypes {
		out[t.Name] = t
	}
	return out
}()
