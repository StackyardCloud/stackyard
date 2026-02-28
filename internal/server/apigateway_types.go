package server

type apiGatewayDataType struct {
	Name string
}

// AWS API Gateway data types sourced from:
// https://docs.aws.amazon.com/apigateway/latest/api/API_Types.html
var apiGatewayDataTypes = []apiGatewayDataType{
	{Name: "AccessLogSettings"},
	{Name: "ApiKey"},
	{Name: "ApiStage"},
	{Name: "Authorizer"},
	{Name: "BasePathMapping"},
	{Name: "CanarySettings"},
	{Name: "ClientCertificate"},
	{Name: "Deployment"},
	{Name: "DeploymentCanarySettings"},
	{Name: "DocumentationPart"},
	{Name: "DocumentationPartLocation"},
	{Name: "DocumentationVersion"},
	{Name: "DomainName"},
	{Name: "DomainNameAccessAssociation"},
	{Name: "EndpointConfiguration"},
	{Name: "GatewayResponse"},
	{Name: "Integration"},
	{Name: "IntegrationResponse"},
	{Name: "Method"},
	{Name: "MethodResponse"},
	{Name: "MethodSetting"},
	{Name: "MethodSnapshot"},
	{Name: "Model"},
	{Name: "MutualTlsAuthentication"},
	{Name: "MutualTlsAuthenticationInput"},
	{Name: "PatchOperation"},
	{Name: "QuotaSettings"},
	{Name: "RequestValidator"},
	{Name: "Resource"},
	{Name: "RestApi"},
	{Name: "SdkConfigurationProperty"},
	{Name: "SdkType"},
	{Name: "Stage"},
	{Name: "StageKey"},
	{Name: "ThrottleSettings"},
	{Name: "TlsConfig"},
	{Name: "UsagePlan"},
	{Name: "UsagePlanKey"},
	{Name: "VpcLink"},
}

var apiGatewayDataTypeByName = func() map[string]apiGatewayDataType {
	out := make(map[string]apiGatewayDataType, len(apiGatewayDataTypes))
	for _, t := range apiGatewayDataTypes {
		out[t.Name] = t
	}
	return out
}()
