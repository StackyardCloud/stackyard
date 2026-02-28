package server

type migrationHubRefactorSpacesDataType struct {
	Name string
}

// AWS Migration Hub Refactor Spaces data types sourced from:
// https://docs.aws.amazon.com/migrationhub-refactor-spaces/latest/APIReference/API_Types.html
var migrationHubRefactorSpacesDataTypes = []migrationHubRefactorSpacesDataType{
	{Name: "ApiGatewayProxyConfig"},
	{Name: "ApiGatewayProxyInput"},
	{Name: "ApiGatewayProxySummary"},
	{Name: "ApplicationSummary"},
	{Name: "DefaultRouteInput"},
	{Name: "EnvironmentSummary"},
	{Name: "EnvironmentVpc"},
	{Name: "ErrorResponse"},
	{Name: "LambdaEndpointConfig"},
	{Name: "LambdaEndpointInput"},
	{Name: "LambdaEndpointSummary"},
	{Name: "RouteSummary"},
	{Name: "ServiceSummary"},
	{Name: "UpdateRoute"},
	{Name: "UriPathRouteInput"},
	{Name: "UrlEndpointConfig"},
	{Name: "UrlEndpointInput"},
	{Name: "UrlEndpointSummary"},
}

var migrationHubRefactorSpacesDataTypeByName = func() map[string]migrationHubRefactorSpacesDataType {
	out := make(map[string]migrationHubRefactorSpacesDataType, len(migrationHubRefactorSpacesDataTypes))
	for _, dt := range migrationHubRefactorSpacesDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
