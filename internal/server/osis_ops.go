package server

type osisOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon OpenSearch Ingestion operations sourced from:
// https://docs.aws.amazon.com/opensearch-service/latest/APIReference/API_Operations_Amazon_OpenSearch_Ingestion.html
var osisOperations = []osisOperation{
	{Name: "CreatePipeline", Method: "POST", URI: "/2022-01-01/osis/createPipeline"},
	{Name: "CreatePipelineEndpoint", Method: "POST", URI: "/2022-01-01/osis/createPipelineEndpoint"},
	{Name: "DeletePipeline", Method: "DELETE", URI: "/2022-01-01/osis/deletePipeline/{PipelineName}"},
	{Name: "DeletePipelineEndpoint", Method: "DELETE", URI: "/2022-01-01/osis/deletePipelineEndpoint/{EndpointId}"},
	{Name: "DeleteResourcePolicy", Method: "DELETE", URI: "/2022-01-01/osis/resourcePolicy/{ResourceArn}"},
	{Name: "GetPipeline", Method: "GET", URI: "/2022-01-01/osis/getPipeline/{PipelineName}"},
	{Name: "GetPipelineBlueprint", Method: "GET", URI: "/2022-01-01/osis/getPipelineBlueprint/{BlueprintName}"},
	{Name: "GetPipelineChangeProgress", Method: "GET", URI: "/2022-01-01/osis/getPipelineChangeProgress/{PipelineName}"},
	{Name: "GetResourcePolicy", Method: "GET", URI: "/2022-01-01/osis/resourcePolicy/{ResourceArn}"},
	{Name: "ListPipelineBlueprints", Method: "POST", URI: "/2022-01-01/osis/listPipelineBlueprints"},
	{Name: "ListPipelineEndpointConnections", Method: "GET", URI: "/2022-01-01/osis/listPipelineEndpointConnections"},
	{Name: "ListPipelineEndpoints", Method: "GET", URI: "/2022-01-01/osis/listPipelineEndpoints"},
	{Name: "ListPipelines", Method: "GET", URI: "/2022-01-01/osis/listPipelines"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/2022-01-01/osis/listTagsForResource/"},
	{Name: "PutResourcePolicy", Method: "PUT", URI: "/2022-01-01/osis/resourcePolicy/{ResourceArn}"},
	{Name: "RevokePipelineEndpointConnections", Method: "POST", URI: "/2022-01-01/osis/revokePipelineEndpointConnections"},
	{Name: "StartPipeline", Method: "PUT", URI: "/2022-01-01/osis/startPipeline/{PipelineName}"},
	{Name: "StopPipeline", Method: "PUT", URI: "/2022-01-01/osis/stopPipeline/{PipelineName}"},
	{Name: "TagResource", Method: "POST", URI: "/2022-01-01/osis/tagResource/"},
	{Name: "UntagResource", Method: "POST", URI: "/2022-01-01/osis/untagResource/"},
	{Name: "UpdatePipeline", Method: "PUT", URI: "/2022-01-01/osis/updatePipeline/{PipelineName}"},
	{Name: "ValidatePipeline", Method: "POST", URI: "/2022-01-01/osis/validatePipeline"},
}

var osisOperationByName = func() map[string]osisOperation {
	out := make(map[string]osisOperation, len(osisOperations))
	for _, op := range osisOperations {
		out[op.Name] = op
	}
	return out
}()
