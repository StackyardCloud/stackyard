package server

type supplyChainOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Supply Chain actions sourced from:
// https://docs.aws.amazon.com/aws-supply-chain/latest/APIReference/API_Operations.html
var supplyChainOperations = []supplyChainOperation{
	{Name: "CreateBillOfMaterialsImportJob", Method: "POST", URI: "/api/configuration/instances/{instanceId}/bill-of-materials-import-jobs"},
	{Name: "CreateDataIntegrationFlow", Method: "PUT", URI: "/api/data-integration/instance/{instanceId}/data-integration-flows/{name}"},
	{Name: "CreateDataLakeDataset", Method: "PUT", URI: "/api/datalake/instance/{instanceId}/namespaces/{namespace}/datasets/{name}"},
	{Name: "CreateDataLakeNamespace", Method: "PUT", URI: "/api/datalake/instance/{instanceId}/namespaces/{name}"},
	{Name: "CreateInstance", Method: "POST", URI: "/api/instance"},
	{Name: "DeleteDataIntegrationFlow", Method: "DELETE", URI: "/api/data-integration/instance/{instanceId}/data-integration-flows/{name}"},
	{Name: "DeleteDataLakeDataset", Method: "DELETE", URI: "/api/datalake/instance/{instanceId}/namespaces/{namespace}/datasets/{name}"},
	{Name: "DeleteDataLakeNamespace", Method: "DELETE", URI: "/api/datalake/instance/{instanceId}/namespaces/{name}"},
	{Name: "DeleteInstance", Method: "DELETE", URI: "/api/instance/{instanceId}"},
	{Name: "GetBillOfMaterialsImportJob", Method: "GET", URI: "/api/configuration/instances/{instanceId}/bill-of-materials-import-jobs/{jobId}"},
	{Name: "GetDataIntegrationEvent", Method: "GET", URI: "/api-data/data-integration/instance/{instanceId}/data-integration-events/{eventId}"},
	{Name: "GetDataIntegrationFlow", Method: "GET", URI: "/api/data-integration/instance/{instanceId}/data-integration-flows/{name}"},
	{Name: "GetDataIntegrationFlowExecution", Method: "GET", URI: "/api-data/data-integration/instance/{instanceId}/data-integration-flows/{flowName}/executions/{executionId}"},
	{Name: "GetDataLakeDataset", Method: "GET", URI: "/api/datalake/instance/{instanceId}/namespaces/{namespace}/datasets/{name}"},
	{Name: "GetDataLakeNamespace", Method: "GET", URI: "/api/datalake/instance/{instanceId}/namespaces/{name}"},
	{Name: "GetInstance", Method: "GET", URI: "/api/instance/{instanceId}"},
	{Name: "ListDataIntegrationEvents", Method: "GET", URI: "/api-data/data-integration/instance/{instanceId}/data-integration-events?eventType={eventType}&maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListDataIntegrationFlowExecutions", Method: "GET", URI: "/api-data/data-integration/instance/{instanceId}/data-integration-flows/{flowName}/executions?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListDataIntegrationFlows", Method: "GET", URI: "/api/data-integration/instance/{instanceId}/data-integration-flows?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListDataLakeDatasets", Method: "GET", URI: "/api/datalake/instance/{instanceId}/namespaces/{namespace}/datasets?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListDataLakeNamespaces", Method: "GET", URI: "/api/datalake/instance/{instanceId}/namespaces?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListInstances", Method: "GET", URI: "/api/instance?instanceNameFilter={instanceNameFilter}&instanceStateFilter={instanceStateFilter}&maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/api/tags/{resourceArn}"},
	{Name: "SendDataIntegrationEvent", Method: "POST", URI: "/api-data/data-integration/instance/{instanceId}/data-integration-events"},
	{Name: "TagResource", Method: "POST", URI: "/api/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/api/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateDataIntegrationFlow", Method: "PATCH", URI: "/api/data-integration/instance/{instanceId}/data-integration-flows/{name}"},
	{Name: "UpdateDataLakeDataset", Method: "PATCH", URI: "/api/datalake/instance/{instanceId}/namespaces/{namespace}/datasets/{name}"},
	{Name: "UpdateDataLakeNamespace", Method: "PATCH", URI: "/api/datalake/instance/{instanceId}/namespaces/{name}"},
	{Name: "UpdateInstance", Method: "PATCH", URI: "/api/instance/{instanceId}"},
}

var supplyChainOperationByName = func() map[string]supplyChainOperation {
	out := make(map[string]supplyChainOperation, len(supplyChainOperations))
	for _, op := range supplyChainOperations {
		out[op.Name] = op
	}
	return out
}()
