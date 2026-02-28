package server

type appSyncOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS AppSync actions sourced from:
// https://docs.aws.amazon.com/appsync/latest/APIReference/API_Operations.html
var appSyncOperations = []appSyncOperation{
	{Name: "AssociateApi", Method: "POST", URI: "/v1/domainnames/{domainName}/apiassociation"},
	{Name: "AssociateMergedGraphqlApi", Method: "POST", URI: "/v1/sourceApis/{sourceApiIdentifier}/mergedApiAssociations"},
	{Name: "AssociateSourceGraphqlApi", Method: "POST", URI: "/v1/mergedApis/{mergedApiIdentifier}/sourceApiAssociations"},
	{Name: "CreateApi", Method: "POST", URI: "/v2/apis"},
	{Name: "CreateApiCache", Method: "POST", URI: "/v1/apis/{apiId}/ApiCaches"},
	{Name: "CreateApiKey", Method: "POST", URI: "/v1/apis/{apiId}/apikeys"},
	{Name: "CreateChannelNamespace", Method: "POST", URI: "/v2/apis/{apiId}/channelNamespaces"},
	{Name: "CreateDataSource", Method: "POST", URI: "/v1/apis/{apiId}/datasources"},
	{Name: "CreateDomainName", Method: "POST", URI: "/v1/domainnames"},
	{Name: "CreateFunction", Method: "POST", URI: "/v1/apis/{apiId}/functions"},
	{Name: "CreateGraphqlApi", Method: "POST", URI: "/v1/apis"},
	{Name: "CreateResolver", Method: "POST", URI: "/v1/apis/{apiId}/types/{typeName}/resolvers"},
	{Name: "CreateType", Method: "POST", URI: "/v1/apis/{apiId}/types"},
	{Name: "DeleteApi", Method: "DELETE", URI: "/v2/apis/{apiId}"},
	{Name: "DeleteApiCache", Method: "DELETE", URI: "/v1/apis/{apiId}/ApiCaches"},
	{Name: "DeleteApiKey", Method: "DELETE", URI: "/v1/apis/{apiId}/apikeys/{id}"},
	{Name: "DeleteChannelNamespace", Method: "DELETE", URI: "/v2/apis/{apiId}/channelNamespaces/{name}"},
	{Name: "DeleteDataSource", Method: "DELETE", URI: "/v1/apis/{apiId}/datasources/{name}"},
	{Name: "DeleteDomainName", Method: "DELETE", URI: "/v1/domainnames/{domainName}"},
	{Name: "DeleteFunction", Method: "DELETE", URI: "/v1/apis/{apiId}/functions/{functionId}"},
	{Name: "DeleteGraphqlApi", Method: "DELETE", URI: "/v1/apis/{apiId}"},
	{Name: "DeleteResolver", Method: "DELETE", URI: "/v1/apis/{apiId}/types/{typeName}/resolvers/{fieldName}"},
	{Name: "DeleteType", Method: "DELETE", URI: "/v1/apis/{apiId}/types/{typeName}"},
	{Name: "DisassociateApi", Method: "DELETE", URI: "/v1/domainnames/{domainName}/apiassociation"},
	{Name: "DisassociateMergedGraphqlApi", Method: "DELETE", URI: "/v1/sourceApis/{sourceApiIdentifier}/mergedApiAssociations/{associationId}"},
	{Name: "DisassociateSourceGraphqlApi", Method: "DELETE", URI: "/v1/mergedApis/{mergedApiIdentifier}/sourceApiAssociations/{associationId}"},
	{Name: "EvaluateCode", Method: "POST", URI: "/v1/dataplane-evaluatecode"},
	{Name: "EvaluateMappingTemplate", Method: "POST", URI: "/v1/dataplane-evaluatetemplate"},
	{Name: "FlushApiCache", Method: "DELETE", URI: "/v1/apis/{apiId}/FlushCache"},
	{Name: "GetApi", Method: "GET", URI: "/v2/apis/{apiId}"},
	{Name: "GetApiAssociation", Method: "GET", URI: "/v1/domainnames/{domainName}/apiassociation"},
	{Name: "GetApiCache", Method: "GET", URI: "/v1/apis/{apiId}/ApiCaches"},
	{Name: "GetChannelNamespace", Method: "GET", URI: "/v2/apis/{apiId}/channelNamespaces/{name}"},
	{Name: "GetDataSource", Method: "GET", URI: "/v1/apis/{apiId}/datasources/{name}"},
	{Name: "GetDataSourceIntrospection", Method: "GET", URI: "/v1/datasources/introspections/{introspectionId}"},
	{Name: "GetDomainName", Method: "GET", URI: "/v1/domainnames/{domainName}"},
	{Name: "GetFunction", Method: "GET", URI: "/v1/apis/{apiId}/functions/{functionId}"},
	{Name: "GetGraphqlApi", Method: "GET", URI: "/v1/apis/{apiId}"},
	{Name: "GetGraphqlApiEnvironmentVariables", Method: "GET", URI: "/v1/apis/{apiId}/environmentVariables"},
	{Name: "GetIntrospectionSchema", Method: "GET", URI: "/v1/apis/{apiId}/schema"},
	{Name: "GetResolver", Method: "GET", URI: "/v1/apis/{apiId}/types/{typeName}/resolvers/{fieldName}"},
	{Name: "GetSchemaCreationStatus", Method: "GET", URI: "/v1/apis/{apiId}/schemacreation"},
	{Name: "GetSourceApiAssociation", Method: "GET", URI: "/v1/mergedApis/{mergedApiIdentifier}/sourceApiAssociations/{associationId}"},
	{Name: "GetType", Method: "GET", URI: "/v1/apis/{apiId}/types/{typeName}"},
	{Name: "ListApiKeys", Method: "GET", URI: "/v1/apis/{apiId}/apikeys"},
	{Name: "ListApis", Method: "GET", URI: "/v2/apis"},
	{Name: "ListChannelNamespaces", Method: "GET", URI: "/v2/apis/{apiId}/channelNamespaces"},
	{Name: "ListDataSources", Method: "GET", URI: "/v1/apis/{apiId}/datasources"},
	{Name: "ListDomainNames", Method: "GET", URI: "/v1/domainnames"},
	{Name: "ListFunctions", Method: "GET", URI: "/v1/apis/{apiId}/functions"},
	{Name: "ListGraphqlApis", Method: "GET", URI: "/v1/apis"},
	{Name: "ListResolvers", Method: "GET", URI: "/v1/apis/{apiId}/types/{typeName}/resolvers"},
	{Name: "ListResolversByFunction", Method: "GET", URI: "/v1/apis/{apiId}/functions/{functionId}/resolvers"},
	{Name: "ListSourceApiAssociations", Method: "GET", URI: "/v1/apis/{apiId}/sourceApiAssociations"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/v1/tags/{resourceArn}"},
	{Name: "ListTypes", Method: "GET", URI: "/v1/apis/{apiId}/types"},
	{Name: "ListTypesByAssociation", Method: "GET", URI: "/v1/mergedApis/{mergedApiIdentifier}/sourceApiAssociations/{associationId}/types"},
	{Name: "PutGraphqlApiEnvironmentVariables", Method: "PUT", URI: "/v1/apis/{apiId}/environmentVariables"},
	{Name: "StartDataSourceIntrospection", Method: "POST", URI: "/v1/datasources/introspections"},
	{Name: "StartSchemaCreation", Method: "POST", URI: "/v1/apis/{apiId}/schemacreation"},
	{Name: "StartSchemaMerge", Method: "POST", URI: "/v1/mergedApis/{mergedApiIdentifier}/sourceApiAssociations/{associationId}/merge"},
	{Name: "TagResource", Method: "POST", URI: "/v1/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/v1/tags/{resourceArn}"},
	{Name: "UpdateApi", Method: "POST", URI: "/v2/apis/{apiId}"},
	{Name: "UpdateApiCache", Method: "POST", URI: "/v1/apis/{apiId}/ApiCaches/update"},
	{Name: "UpdateApiKey", Method: "POST", URI: "/v1/apis/{apiId}/apikeys/{id}"},
	{Name: "UpdateChannelNamespace", Method: "POST", URI: "/v2/apis/{apiId}/channelNamespaces/{name}"},
	{Name: "UpdateDataSource", Method: "POST", URI: "/v1/apis/{apiId}/datasources/{name}"},
	{Name: "UpdateDomainName", Method: "POST", URI: "/v1/domainnames/{domainName}"},
	{Name: "UpdateFunction", Method: "POST", URI: "/v1/apis/{apiId}/functions/{functionId}"},
	{Name: "UpdateGraphqlApi", Method: "POST", URI: "/v1/apis/{apiId}"},
	{Name: "UpdateResolver", Method: "POST", URI: "/v1/apis/{apiId}/types/{typeName}/resolvers/{fieldName}"},
	{Name: "UpdateSourceApiAssociation", Method: "POST", URI: "/v1/mergedApis/{mergedApiIdentifier}/sourceApiAssociations/{associationId}"},
	{Name: "UpdateType", Method: "POST", URI: "/v1/apis/{apiId}/types/{typeName}"},
}

var appSyncOperationByName = func() map[string]appSyncOperation {
	out := make(map[string]appSyncOperation, len(appSyncOperations))
	for _, op := range appSyncOperations {
		out[op.Name] = op
	}
	return out
}()
