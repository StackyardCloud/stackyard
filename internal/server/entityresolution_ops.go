package server

type entityResolutionOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Entity Resolution actions sourced from:
// https://docs.aws.amazon.com/entityresolution/latest/apireference/API_Operations.html
var entityResolutionOperations = []entityResolutionOperation{
	{Name: "AddPolicyStatement", Method: "POST", URI: "/policies/{arn}/{statementId}"},
	{Name: "BatchDeleteUniqueId", Method: "DELETE", URI: "/matchingworkflows/{workflowName}/uniqueids"},
	{Name: "CreateIdMappingWorkflow", Method: "POST", URI: "/idmappingworkflows"},
	{Name: "CreateIdNamespace", Method: "POST", URI: "/idnamespaces"},
	{Name: "CreateMatchingWorkflow", Method: "POST", URI: "/matchingworkflows"},
	{Name: "CreateSchemaMapping", Method: "POST", URI: "/schemas"},
	{Name: "DeleteIdMappingWorkflow", Method: "DELETE", URI: "/idmappingworkflows/{workflowName}"},
	{Name: "DeleteIdNamespace", Method: "DELETE", URI: "/idnamespaces/{idNamespaceName}"},
	{Name: "DeleteMatchingWorkflow", Method: "DELETE", URI: "/matchingworkflows/{workflowName}"},
	{Name: "DeletePolicyStatement", Method: "DELETE", URI: "/policies/{arn}/{statementId}"},
	{Name: "DeleteSchemaMapping", Method: "DELETE", URI: "/schemas/{schemaName}"},
	{Name: "GenerateMatchId", Method: "POST", URI: "/matchingworkflows/{workflowName}/generateMatches"},
	{Name: "GetIdMappingJob", Method: "GET", URI: "/idmappingworkflows/{workflowName}/jobs/{jobId}"},
	{Name: "GetIdMappingWorkflow", Method: "GET", URI: "/idmappingworkflows/{workflowName}"},
	{Name: "GetIdNamespace", Method: "GET", URI: "/idnamespaces/{idNamespaceName}"},
	{Name: "GetMatchId", Method: "POST", URI: "/matchingworkflows/{workflowName}/matches"},
	{Name: "GetMatchingJob", Method: "GET", URI: "/matchingworkflows/{workflowName}/jobs/{jobId}"},
	{Name: "GetMatchingWorkflow", Method: "GET", URI: "/matchingworkflows/{workflowName}"},
	{Name: "GetPolicy", Method: "GET", URI: "/policies/{arn}"},
	{Name: "GetProviderService", Method: "GET", URI: "/providerservices/{providerName}/{providerServiceName}"},
	{Name: "GetSchemaMapping", Method: "GET", URI: "/schemas/{schemaName}"},
	{Name: "ListIdMappingJobs", Method: "GET", URI: "/idmappingworkflows/{workflowName}/jobs?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListIdMappingWorkflows", Method: "GET", URI: "/idmappingworkflows?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListIdNamespaces", Method: "GET", URI: "/idnamespaces?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListMatchingJobs", Method: "GET", URI: "/matchingworkflows/{workflowName}/jobs?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListMatchingWorkflows", Method: "GET", URI: "/matchingworkflows?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListProviderServices", Method: "GET", URI: "/providerservices?maxResults={maxResults}&nextToken={nextToken}&providerName={providerName}"},
	{Name: "ListSchemaMappings", Method: "GET", URI: "/schemas?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "PutPolicy", Method: "PUT", URI: "/policies/{arn}"},
	{Name: "StartIdMappingJob", Method: "POST", URI: "/idmappingworkflows/{workflowName}/jobs"},
	{Name: "StartMatchingJob", Method: "POST", URI: "/matchingworkflows/{workflowName}/jobs"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateIdMappingWorkflow", Method: "PUT", URI: "/idmappingworkflows/{workflowName}"},
	{Name: "UpdateIdNamespace", Method: "PUT", URI: "/idnamespaces/{idNamespaceName}"},
	{Name: "UpdateMatchingWorkflow", Method: "PUT", URI: "/matchingworkflows/{workflowName}"},
	{Name: "UpdateSchemaMapping", Method: "PUT", URI: "/schemas/{schemaName}"},
}

var entityResolutionOperationByName = func() map[string]entityResolutionOperation {
	out := make(map[string]entityResolutionOperation, len(entityResolutionOperations))
	for _, op := range entityResolutionOperations {
		out[op.Name] = op
	}
	return out
}()
