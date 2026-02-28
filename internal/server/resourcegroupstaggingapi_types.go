package server

type resourceGroupsTaggingAPIDataType struct {
	Name string
}

// AWS Resource Groups Tagging API data types sourced from:
// https://docs.aws.amazon.com/resourcegroupstagging/latest/APIReference/API_Types.html
var resourceGroupsTaggingAPIDataTypes = []resourceGroupsTaggingAPIDataType{
	{Name: "ComplianceDetails"},
	{Name: "FailureInfo"},
	{Name: "ResourceTagMapping"},
	{Name: "Summary"},
	{Name: "Tag"},
	{Name: "TagFilter"},
}

var resourceGroupsTaggingAPIDataTypeByName = func() map[string]resourceGroupsTaggingAPIDataType {
	out := make(map[string]resourceGroupsTaggingAPIDataType, len(resourceGroupsTaggingAPIDataTypes))
	for _, dt := range resourceGroupsTaggingAPIDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
