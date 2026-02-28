package server

type resourceExplorer2DataType struct {
	Name string
}

// AWS Resource Explorer data types sourced from:
// https://docs.aws.amazon.com/resource-explorer/latest/apireference/API_Types.html
var resourceExplorer2DataTypes = []resourceExplorer2DataType{
	{Name: "BatchGetViewError"},
	{Name: "ErrorDetails"},
	{Name: "IncludedProperty"},
	{Name: "Index"},
	{Name: "IndexStatus"},
	{Name: "ManagedView"},
	{Name: "MemberIndex"},
	{Name: "OrgConfiguration"},
	{Name: "RegionStatus"},
	{Name: "Resource"},
	{Name: "ResourceCount"},
	{Name: "ResourceProperty"},
	{Name: "SearchFilter"},
	{Name: "ServiceView"},
	{Name: "StreamingAccessDetails"},
	{Name: "SupportedResourceType"},
	{Name: "UpdateView"},
	{Name: "ValidationExceptionField"},
	{Name: "View"},
	{Name: "ViewStatus"},
}

var resourceExplorer2DataTypeByName = func() map[string]resourceExplorer2DataType {
	out := make(map[string]resourceExplorer2DataType, len(resourceExplorer2DataTypes))
	for _, dt := range resourceExplorer2DataTypes {
		out[dt.Name] = dt
	}
	return out
}()
