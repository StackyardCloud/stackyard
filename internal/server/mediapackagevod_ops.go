package server

type mpvodOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Elemental MediaPackage VOD operations sourced from:
// https://docs.aws.amazon.com/mediapackage-vod/latest/apireference/resources.html
var mpvodOperations = []mpvodOperation{
	{Name: "ConfigureLogs", Method: "PUT", URI: "/packaging_groups/{id}/configure_logs"},
	{Name: "CreateAsset", Method: "POST", URI: "/assets"},
	{Name: "CreatePackagingConfiguration", Method: "POST", URI: "/packaging_configurations"},
	{Name: "CreatePackagingGroup", Method: "POST", URI: "/packaging_groups"},
	{Name: "DeleteAsset", Method: "DELETE", URI: "/assets/{id}"},
	{Name: "DeletePackagingConfiguration", Method: "DELETE", URI: "/packaging_configurations/{id}"},
	{Name: "DeletePackagingGroup", Method: "DELETE", URI: "/packaging_groups/{id}"},
	{Name: "DescribeAsset", Method: "GET", URI: "/assets/{id}"},
	{Name: "DescribePackagingConfiguration", Method: "GET", URI: "/packaging_configurations/{id}"},
	{Name: "DescribePackagingGroup", Method: "GET", URI: "/packaging_groups/{id}"},
	{Name: "ListAssets", Method: "GET", URI: "/assets"},
	{Name: "ListPackagingConfigurations", Method: "GET", URI: "/packaging_configurations"},
	{Name: "ListPackagingGroups", Method: "GET", URI: "/packaging_groups"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resource-arn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resource-arn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resource-arn}"},
	{Name: "UpdatePackagingGroup", Method: "PUT", URI: "/packaging_groups/{id}"},
}

var mpvodOperationByName = func() map[string]mpvodOperation {
	out := make(map[string]mpvodOperation, len(mpvodOperations))
	for _, op := range mpvodOperations {
		out[op.Name] = op
	}
	return out
}()
