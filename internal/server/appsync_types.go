package server

type appSyncDataType struct {
	Name string
}

// AWS AppSync data types sourced from:
// https://docs.aws.amazon.com/appsync/latest/APIReference/API_Types.html
var appSyncDataTypes = []appSyncDataType{
	{Name: "AdditionalAuthenticationProvider"},
	{Name: "Api"},
	{Name: "ApiAssociation"},
	{Name: "ApiCache"},
	{Name: "ApiKey"},
	{Name: "AppSyncRuntime"},
	{Name: "AuthMode"},
	{Name: "AuthorizationConfig"},
	{Name: "AuthProvider"},
	{Name: "AwsIamConfig"},
	{Name: "BadRequestDetail"},
	{Name: "CachingConfig"},
	{Name: "ChannelNamespace"},
	{Name: "CodeError"},
	{Name: "CodeErrorLocation"},
	{Name: "CognitoConfig"},
	{Name: "CognitoUserPoolConfig"},
	{Name: "DataSource"},
	{Name: "DataSourceIntrospectionModel"},
	{Name: "DataSourceIntrospectionModelField"},
	{Name: "DataSourceIntrospectionModelFieldType"},
	{Name: "DataSourceIntrospectionModelIndex"},
	{Name: "DataSourceIntrospectionResult"},
	{Name: "DeltaSyncConfig"},
	{Name: "DomainNameConfig"},
	{Name: "DynamodbDataSourceConfig"},
	{Name: "ElasticsearchDataSourceConfig"},
	{Name: "EnhancedMetricsConfig"},
	{Name: "ErrorDetail"},
	{Name: "EvaluateCodeErrorDetail"},
	{Name: "EventBridgeDataSourceConfig"},
	{Name: "EventConfig"},
	{Name: "EventLogConfig"},
	{Name: "FunctionConfiguration"},
	{Name: "GraphqlApi"},
	{Name: "HandlerConfig"},
	{Name: "HandlerConfigs"},
	{Name: "HttpDataSourceConfig"},
	{Name: "Integration"},
	{Name: "LambdaAuthorizerConfig"},
	{Name: "LambdaConfig"},
	{Name: "LambdaConflictHandlerConfig"},
	{Name: "LambdaDataSourceConfig"},
	{Name: "LogConfig"},
	{Name: "OpenIDConnectConfig"},
	{Name: "OpenSearchServiceDataSourceConfig"},
	{Name: "PipelineConfig"},
	{Name: "RdsDataApiConfig"},
	{Name: "RdsHttpEndpointConfig"},
	{Name: "RelationalDatabaseDataSourceConfig"},
	{Name: "Resolver"},
	{Name: "SourceApiAssociation"},
	{Name: "SourceApiAssociationConfig"},
	{Name: "SourceApiAssociationSummary"},
	{Name: "SyncConfig"},
	{Name: "Type"},
	{Name: "UserPoolConfig"},
}

var appSyncDataTypeByName = func() map[string]appSyncDataType {
	out := make(map[string]appSyncDataType, len(appSyncDataTypes))
	for _, dt := range appSyncDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
