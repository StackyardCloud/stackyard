package server

type lakeFormationOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Lake Formation actions sourced from:
// https://docs.aws.amazon.com/lake-formation/latest/APIReference/API_Operations.html
var lakeFormationOperations = []lakeFormationOperation{
	{Name: "AddLFTagsToResource", Method: "POST", URI: "/AddLFTagsToResource"},
	{Name: "AssumeDecoratedRoleWithSAML", Method: "POST", URI: "/AssumeDecoratedRoleWithSAML"},
	{Name: "BatchGrantPermissions", Method: "POST", URI: "/BatchGrantPermissions"},
	{Name: "BatchRevokePermissions", Method: "POST", URI: "/BatchRevokePermissions"},
	{Name: "CancelTransaction", Method: "POST", URI: "/CancelTransaction"},
	{Name: "CommitTransaction", Method: "POST", URI: "/CommitTransaction"},
	{Name: "CreateDataCellsFilter", Method: "POST", URI: "/CreateDataCellsFilter"},
	{Name: "CreateLFTag", Method: "POST", URI: "/CreateLFTag"},
	{Name: "CreateLFTagExpression", Method: "POST", URI: "/CreateLFTagExpression"},
	{Name: "CreateLakeFormationIdentityCenterConfiguration", Method: "POST", URI: "/CreateLakeFormationIdentityCenterConfiguration"},
	{Name: "CreateLakeFormationOptIn", Method: "POST", URI: "/CreateLakeFormationOptIn"},
	{Name: "DeleteDataCellsFilter", Method: "POST", URI: "/DeleteDataCellsFilter"},
	{Name: "DeleteLFTag", Method: "POST", URI: "/DeleteLFTag"},
	{Name: "DeleteLFTagExpression", Method: "POST", URI: "/DeleteLFTagExpression"},
	{Name: "DeleteLakeFormationIdentityCenterConfiguration", Method: "POST", URI: "/DeleteLakeFormationIdentityCenterConfiguration"},
	{Name: "DeleteLakeFormationOptIn", Method: "POST", URI: "/DeleteLakeFormationOptIn"},
	{Name: "DeleteObjectsOnCancel", Method: "POST", URI: "/DeleteObjectsOnCancel"},
	{Name: "DeregisterResource", Method: "POST", URI: "/DeregisterResource"},
	{Name: "DescribeLakeFormationIdentityCenterConfiguration", Method: "POST", URI: "/DescribeLakeFormationIdentityCenterConfiguration"},
	{Name: "DescribeResource", Method: "POST", URI: "/DescribeResource"},
	{Name: "DescribeTransaction", Method: "POST", URI: "/DescribeTransaction"},
	{Name: "ExtendTransaction", Method: "POST", URI: "/ExtendTransaction"},
	{Name: "GetDataCellsFilter", Method: "POST", URI: "/GetDataCellsFilter"},
	{Name: "GetDataLakePrincipal", Method: "POST", URI: "/GetDataLakePrincipal"},
	{Name: "GetDataLakeSettings", Method: "POST", URI: "/GetDataLakeSettings"},
	{Name: "GetEffectivePermissionsForPath", Method: "POST", URI: "/GetEffectivePermissionsForPath"},
	{Name: "GetLFTag", Method: "POST", URI: "/GetLFTag"},
	{Name: "GetLFTagExpression", Method: "POST", URI: "/GetLFTagExpression"},
	{Name: "GetQueryState", Method: "POST", URI: "/GetQueryState"},
	{Name: "GetQueryStatistics", Method: "POST", URI: "/GetQueryStatistics"},
	{Name: "GetResourceLFTags", Method: "POST", URI: "/GetResourceLFTags"},
	{Name: "GetTableObjects", Method: "POST", URI: "/GetTableObjects"},
	{Name: "GetTemporaryDataLocationCredentials", Method: "POST", URI: "/GetTemporaryDataLocationCredentials"},
	{Name: "GetTemporaryGluePartitionCredentials", Method: "POST", URI: "/GetTemporaryGluePartitionCredentials"},
	{Name: "GetTemporaryGlueTableCredentials", Method: "POST", URI: "/GetTemporaryGlueTableCredentials"},
	{Name: "GetWorkUnitResults", Method: "POST", URI: "/GetWorkUnitResults"},
	{Name: "GetWorkUnits", Method: "POST", URI: "/GetWorkUnits"},
	{Name: "GrantPermissions", Method: "POST", URI: "/GrantPermissions"},
	{Name: "ListDataCellsFilter", Method: "POST", URI: "/ListDataCellsFilter"},
	{Name: "ListLFTagExpressions", Method: "POST", URI: "/ListLFTagExpressions"},
	{Name: "ListLFTags", Method: "POST", URI: "/ListLFTags"},
	{Name: "ListLakeFormationOptIns", Method: "POST", URI: "/ListLakeFormationOptIns"},
	{Name: "ListPermissions", Method: "POST", URI: "/ListPermissions"},
	{Name: "ListResources", Method: "POST", URI: "/ListResources"},
	{Name: "ListTableStorageOptimizers", Method: "POST", URI: "/ListTableStorageOptimizers"},
	{Name: "ListTransactions", Method: "POST", URI: "/ListTransactions"},
	{Name: "PutDataLakeSettings", Method: "POST", URI: "/PutDataLakeSettings"},
	{Name: "RegisterResource", Method: "POST", URI: "/RegisterResource"},
	{Name: "RemoveLFTagsFromResource", Method: "POST", URI: "/RemoveLFTagsFromResource"},
	{Name: "RevokePermissions", Method: "POST", URI: "/RevokePermissions"},
	{Name: "SearchDatabasesByLFTags", Method: "POST", URI: "/SearchDatabasesByLFTags"},
	{Name: "SearchTablesByLFTags", Method: "POST", URI: "/SearchTablesByLFTags"},
	{Name: "StartQueryPlanning", Method: "POST", URI: "/StartQueryPlanning"},
	{Name: "StartTransaction", Method: "POST", URI: "/StartTransaction"},
	{Name: "UpdateDataCellsFilter", Method: "POST", URI: "/UpdateDataCellsFilter"},
	{Name: "UpdateLFTag", Method: "POST", URI: "/UpdateLFTag"},
	{Name: "UpdateLFTagExpression", Method: "POST", URI: "/UpdateLFTagExpression"},
	{Name: "UpdateLakeFormationIdentityCenterConfiguration", Method: "POST", URI: "/UpdateLakeFormationIdentityCenterConfiguration"},
	{Name: "UpdateResource", Method: "POST", URI: "/UpdateResource"},
	{Name: "UpdateTableObjects", Method: "POST", URI: "/UpdateTableObjects"},
	{Name: "UpdateTableStorageOptimizer", Method: "POST", URI: "/UpdateTableStorageOptimizer"},
}

var lakeFormationOperationByName = func() map[string]lakeFormationOperation {
	out := make(map[string]lakeFormationOperation, len(lakeFormationOperations))
	for _, op := range lakeFormationOperations {
		out[op.Name] = op
	}
	return out
}()
