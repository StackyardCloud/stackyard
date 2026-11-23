package server

type schemasOperation struct {
	Name   string
	Method string
	URI    string
}

// EventBridge Schemas operations sourced from:
// https://docs.aws.amazon.com/eventbridge/latest/schema-reference/operations.html
var schemasOperations = []schemasOperation{
	{Name: "CreateDiscoverer", Method: "POST", URI: "/v1/discoverers"},
	{Name: "CreateRegistry", Method: "POST", URI: "/v1/registries/name/{registryName}"},
	{Name: "CreateSchema", Method: "POST", URI: "/v1/registries/name/{registryName}/schemas/name/{schemaName}"},
	{Name: "DeleteDiscoverer", Method: "DELETE", URI: "/v1/discoverers/id/{discovererId}"},
	{Name: "DeleteRegistry", Method: "DELETE", URI: "/v1/registries/name/{registryName}"},
	{Name: "DeleteResourcePolicy", Method: "DELETE", URI: "/v1/policy"},
	{Name: "DeleteSchema", Method: "DELETE", URI: "/v1/registries/name/{registryName}/schemas/name/{schemaName}"},
	{Name: "DeleteSchemaVersion", Method: "DELETE", URI: "/v1/registries/name/{registryName}/schemas/name/{schemaName}/version/{schemaVersion}"},
	{Name: "DescribeCodeBinding", Method: "GET", URI: "/v1/registries/name/{registryName}/schemas/name/{schemaName}/language/{language}"},
	{Name: "DescribeDiscoverer", Method: "GET", URI: "/v1/discoverers/id/{discovererId}"},
	{Name: "DescribeRegistry", Method: "GET", URI: "/v1/registries/name/{registryName}"},
	{Name: "DescribeSchema", Method: "GET", URI: "/v1/registries/name/{registryName}/schemas/name/{schemaName}"},
	{Name: "ExportSchema", Method: "GET", URI: "/v1/registries/name/{registryName}/schemas/name/{schemaName}/export"},
	{Name: "GetCodeBindingSource", Method: "GET", URI: "/v1/registries/name/{registryName}/schemas/name/{schemaName}/language/{language}/source"},
	{Name: "GetDiscoveredSchema", Method: "POST", URI: "/v1/discover"},
	{Name: "GetResourcePolicy", Method: "GET", URI: "/v1/policy"},
	{Name: "ListDiscoverers", Method: "GET", URI: "/v1/discoverers"},
	{Name: "ListRegistries", Method: "GET", URI: "/v1/registries"},
	{Name: "ListSchemaVersions", Method: "GET", URI: "/v1/registries/name/{registryName}/schemas/name/{schemaName}/versions"},
	{Name: "ListSchemas", Method: "GET", URI: "/v1/registries/name/{registryName}/schemas"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resource-arn}"},
	{Name: "PutCodeBinding", Method: "POST", URI: "/v1/registries/name/{registryName}/schemas/name/{schemaName}/language/{language}"},
	{Name: "PutResourcePolicy", Method: "PUT", URI: "/v1/policy"},
	{Name: "SearchSchemas", Method: "GET", URI: "/v1/registries/name/{registryName}/schemas/search"},
	{Name: "StartDiscoverer", Method: "POST", URI: "/v1/discoverers/id/{discovererId}/start"},
	{Name: "StopDiscoverer", Method: "POST", URI: "/v1/discoverers/id/{discovererId}/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resource-arn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resource-arn}"},
	{Name: "UpdateDiscoverer", Method: "PUT", URI: "/v1/discoverers/id/{discovererId}"},
	{Name: "UpdateRegistry", Method: "PUT", URI: "/v1/registries/name/{registryName}"},
	{Name: "UpdateSchema", Method: "PUT", URI: "/v1/registries/name/{registryName}/schemas/name/{schemaName}"},
}

var schemasOperationByName = func() map[string]schemasOperation {
	out := make(map[string]schemasOperation, len(schemasOperations))
	for _, op := range schemasOperations {
		out[op.Name] = op
	}
	return out
}()
