package server

type ramDataType struct {
	Name string
}

// AWS RAM data types sourced from:
// https://docs.aws.amazon.com/ram/latest/APIReference/API_Types.html
var ramDataTypes = []ramDataType{
	{Name: "AssociatedPermission"},
	{Name: "AssociatedSource"},
	{Name: "Principal"},
	{Name: "ReplacePermissionAssociationsWork"},
	{Name: "Resource"},
	{Name: "ResourceShare"},
	{Name: "ResourceShareAssociation"},
	{Name: "ResourceShareInvitation"},
	{Name: "ResourceSharePermissionDetail"},
	{Name: "ResourceSharePermissionSummary"},
	{Name: "ServiceNameAndResourceType"},
	{Name: "Tag"},
	{Name: "TagFilter"},
	{Name: "UpdateResourceShare"},
}

var ramDataTypeByName = func() map[string]ramDataType {
	out := make(map[string]ramDataType, len(ramDataTypes))
	for _, dt := range ramDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
