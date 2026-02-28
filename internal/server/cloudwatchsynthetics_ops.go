package server

type cloudWatchSyntheticsOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon CloudWatch Synthetics operations sourced from:
// https://docs.aws.amazon.com/AmazonSynthetics/latest/APIReference/API_Operations.html
var cloudWatchSyntheticsOperations = []cloudWatchSyntheticsOperation{
	{Name: "AssociateResource", Method: "PATCH", URI: "/group/{groupIdentifier}/associate"},
	{Name: "CreateCanary", Method: "POST", URI: "/canary"},
	{Name: "CreateGroup", Method: "POST", URI: "/group"},
	{Name: "DeleteCanary", Method: "DELETE", URI: "/canary/{name}"},
	{Name: "DeleteGroup", Method: "DELETE", URI: "/group/{groupIdentifier}"},
	{Name: "DescribeCanaries", Method: "POST", URI: "/canaries"},
	{Name: "DescribeCanariesLastRun", Method: "POST", URI: "/canaries/last-run"},
	{Name: "DescribeRuntimeVersions", Method: "POST", URI: "/runtime-versions"},
	{Name: "DisassociateResource", Method: "PATCH", URI: "/group/{groupIdentifier}/disassociate"},
	{Name: "GetCanary", Method: "GET", URI: "/canary/{name}"},
	{Name: "GetCanaryRuns", Method: "POST", URI: "/canary/{name}/runs"},
	{Name: "GetGroup", Method: "GET", URI: "/group/{groupIdentifier}"},
	{Name: "ListAssociatedGroups", Method: "POST", URI: "/resource/{resourceArn}/groups"},
	{Name: "ListGroupResources", Method: "POST", URI: "/group/{groupIdentifier}/resources"},
	{Name: "ListGroups", Method: "POST", URI: "/groups"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "StartCanary", Method: "POST", URI: "/canary/{name}/start"},
	{Name: "StartCanaryDryRun", Method: "POST", URI: "/canary/{name}/dry-run/start"},
	{Name: "StopCanary", Method: "POST", URI: "/canary/{name}/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateCanary", Method: "PATCH", URI: "/canary/{name}"},
}

var cloudWatchSyntheticsOperationByName = func() map[string]cloudWatchSyntheticsOperation {
	out := make(map[string]cloudWatchSyntheticsOperation, len(cloudWatchSyntheticsOperations))
	for _, op := range cloudWatchSyntheticsOperations {
		out[op.Name] = op
	}
	return out
}()
