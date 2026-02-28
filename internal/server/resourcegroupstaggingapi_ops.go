package server

type resourceGroupsTaggingAPIOperation struct {
	Name string
}

// AWS Resource Groups Tagging API operations sourced from:
// https://docs.aws.amazon.com/resourcegroupstagging/latest/APIReference/API_Operations.html
var resourceGroupsTaggingAPIOperations = []resourceGroupsTaggingAPIOperation{
	{Name: "DescribeReportCreation"},
	{Name: "GetComplianceSummary"},
	{Name: "GetResources"},
	{Name: "GetTagKeys"},
	{Name: "GetTagValues"},
	{Name: "StartReportCreation"},
	{Name: "TagResources"},
	{Name: "UntagResources"},
}

var resourceGroupsTaggingAPIOperationByName = func() map[string]resourceGroupsTaggingAPIOperation {
	out := make(map[string]resourceGroupsTaggingAPIOperation, len(resourceGroupsTaggingAPIOperations))
	for _, op := range resourceGroupsTaggingAPIOperations {
		out[op.Name] = op
	}
	return out
}()
