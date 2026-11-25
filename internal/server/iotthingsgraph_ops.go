package server

type iotThingsGraphOperation struct{ Name string }

// AWS IoT Things Graph operations sourced from the AWS CLI 2.42.1 model
// (botocore iotthingsgraph 2018-09-06). AWS retired the service and its API
// reference with it; the last published index was
// https://docs.aws.amazon.com/thingsgraph/latest/APIReference/API_Operations.html
var iotThingsGraphOperations = []iotThingsGraphOperation{
	{Name: "AssociateEntityToThing"},
	{Name: "CreateFlowTemplate"},
	{Name: "CreateSystemInstance"},
	{Name: "CreateSystemTemplate"},
	{Name: "DeleteFlowTemplate"},
	{Name: "DeleteNamespace"},
	{Name: "DeleteSystemInstance"},
	{Name: "DeleteSystemTemplate"},
	{Name: "DeploySystemInstance"},
	{Name: "DeprecateFlowTemplate"},
	{Name: "DeprecateSystemTemplate"},
	{Name: "DescribeNamespace"},
	{Name: "DissociateEntityFromThing"},
	{Name: "GetEntities"},
	{Name: "GetFlowTemplate"},
	{Name: "GetFlowTemplateRevisions"},
	{Name: "GetNamespaceDeletionStatus"},
	{Name: "GetSystemInstance"},
	{Name: "GetSystemTemplate"},
	{Name: "GetSystemTemplateRevisions"},
	{Name: "GetUploadStatus"},
	{Name: "ListFlowExecutionMessages"},
	{Name: "ListTagsForResource"},
	{Name: "SearchEntities"},
	{Name: "SearchFlowExecutions"},
	{Name: "SearchFlowTemplates"},
	{Name: "SearchSystemInstances"},
	{Name: "SearchSystemTemplates"},
	{Name: "SearchThings"},
	{Name: "TagResource"},
	{Name: "UndeploySystemInstance"},
	{Name: "UntagResource"},
	{Name: "UpdateFlowTemplate"},
	{Name: "UpdateSystemTemplate"},
	{Name: "UploadEntityDefinitions"},
}

var iotThingsGraphOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range iotThingsGraphOperations {
		out[op.Name] = true
	}
	return out
}()
