package server

type lakeFormationDataType struct {
	Name string
}

// AWS Lake Formation data types sourced from:
// https://docs.aws.amazon.com/lake-formation/latest/APIReference/API_Types.html
var lakeFormationDataTypes = []lakeFormationDataType{
	{Name: "AddObjectInput"},
	{Name: "AllRowsWildcard"},
	{Name: "AuditContext"},
	{Name: "BatchPermissionsFailureEntry"},
	{Name: "BatchPermissionsRequestEntry"},
	{Name: "CatalogResource"},
	{Name: "ColumnLFTag"},
	{Name: "ColumnWildcard"},
	{Name: "Condition"},
	{Name: "DataCellsFilter"},
	{Name: "DataCellsFilterResource"},
	{Name: "DataLakePrincipal"},
	{Name: "DataLakeSettings"},
	{Name: "DataLocationResource"},
	{Name: "DatabaseResource"},
	{Name: "DeleteObjectInput"},
	{Name: "DetailsMap"},
	{Name: "ErrorDetail"},
	{Name: "ExecutionStatistics"},
	{Name: "ExternalFilteringConfiguration"},
	{Name: "FilterCondition"},
	{Name: "LFTag"},
	{Name: "LFTagError"},
	{Name: "LFTagExpression"},
	{Name: "LFTagExpressionResource"},
	{Name: "LFTagKeyResource"},
	{Name: "LFTagPair"},
	{Name: "LFTagPolicyResource"},
	{Name: "LakeFormationOptInsInfo"},
	{Name: "PartitionObjects"},
	{Name: "PartitionValueList"},
	{Name: "PlanningStatistics"},
	{Name: "PrincipalPermissions"},
	{Name: "PrincipalResourcePermissions"},
	{Name: "QueryPlanningContext"},
	{Name: "QuerySessionContext"},
	{Name: "RedshiftConnect"},
	{Name: "RedshiftScopeUnion"},
	{Name: "Resource"},
	{Name: "ResourceInfo"},
	{Name: "RowFilter"},
	{Name: "ServiceIntegrationUnion"},
	{Name: "StorageOptimizer"},
	{Name: "TableObject"},
	{Name: "TableResource"},
	{Name: "TableWildcard"},
	{Name: "TableWithColumnsResource"},
	{Name: "TaggedDatabase"},
	{Name: "TaggedTable"},
	{Name: "TemporaryCredentials"},
	{Name: "TransactionDescription"},
	{Name: "UpdateTableStorageOptimizer"},
	{Name: "VirtualObject"},
	{Name: "WorkUnitRange"},
	{Name: "WriteOperation"},
}

var lakeFormationDataTypeByName = func() map[string]lakeFormationDataType {
	out := make(map[string]lakeFormationDataType, len(lakeFormationDataTypes))
	for _, t := range lakeFormationDataTypes {
		out[t.Name] = t
	}
	return out
}()
