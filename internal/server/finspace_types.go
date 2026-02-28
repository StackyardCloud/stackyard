package server

type finspaceDataType struct {
	Name string
}

// Amazon FinSpace Data API data types sourced from:
// https://docs.aws.amazon.com/finspace/latest/data-api/API_Types.html
var finspaceDataTypes = []finspaceDataType{
	{Name: "AwsCredentials"},
	{Name: "ChangesetErrorInfo"},
	{Name: "ChangesetSummary"},
	{Name: "ColumnDefinition"},
	{Name: "Credentials"},
	{Name: "Dataset"},
	{Name: "DatasetOwnerInfo"},
	{Name: "DataViewDestinationTypeParams"},
	{Name: "DataViewErrorInfo"},
	{Name: "DataViewSummary"},
	{Name: "PermissionGroup"},
	{Name: "PermissionGroupByUser"},
	{Name: "PermissionGroupParams"},
	{Name: "ResourcePermission"},
	{Name: "S3Location"},
	{Name: "SchemaDefinition"},
	{Name: "SchemaUnion"},
	{Name: "User"},
	{Name: "UserByPermissionGroup"},
	{Name: "UpdateUser"},
}

var finspaceDataTypeByName = func() map[string]finspaceDataType {
	out := make(map[string]finspaceDataType, len(finspaceDataTypes))
	for _, dt := range finspaceDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
