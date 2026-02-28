package server

type supplyChainDataType struct {
	Name string
}

// AWS Supply Chain data types sourced from:
// https://docs.aws.amazon.com/aws-supply-chain/latest/APIReference/API_Types.html
var supplyChainDataTypes = []supplyChainDataType{
	{Name: "BillOfMaterialsImportJob"},
	{Name: "DataIntegrationEvent"},
	{Name: "DataIntegrationEventDatasetLoadExecutionDetails"},
	{Name: "DataIntegrationEventDatasetTargetConfiguration"},
	{Name: "DataIntegrationEventDatasetTargetDetails"},
	{Name: "DataIntegrationFlow"},
	{Name: "DataIntegrationFlowDatasetOptions"},
	{Name: "DataIntegrationFlowDatasetSource"},
	{Name: "DataIntegrationFlowDatasetSourceConfiguration"},
	{Name: "DataIntegrationFlowDatasetTargetConfiguration"},
	{Name: "DataIntegrationFlowDedupeStrategy"},
	{Name: "DataIntegrationFlowExecution"},
	{Name: "DataIntegrationFlowExecutionOutputMetadata"},
	{Name: "DataIntegrationFlowExecutionSourceInfo"},
	{Name: "DataIntegrationFlowFieldPriorityDedupeField"},
	{Name: "DataIntegrationFlowFieldPriorityDedupeStrategyConfiguration"},
	{Name: "DataIntegrationFlowS3Options"},
	{Name: "DataIntegrationFlowS3Source"},
	{Name: "DataIntegrationFlowS3SourceConfiguration"},
	{Name: "DataIntegrationFlowS3TargetConfiguration"},
	{Name: "DataIntegrationFlowSource"},
	{Name: "DataIntegrationFlowSQLTransformationConfiguration"},
	{Name: "DataIntegrationFlowTarget"},
	{Name: "DataIntegrationFlowTransformation"},
	{Name: "DataLakeDataset"},
	{Name: "DataLakeDatasetPartitionField"},
	{Name: "DataLakeDatasetPartitionFieldTransform"},
	{Name: "DataLakeDatasetPartitionSpec"},
	{Name: "DataLakeDatasetPrimaryKeyField"},
	{Name: "DataLakeDatasetSchema"},
	{Name: "DataLakeDatasetSchemaField"},
	{Name: "DataLakeNamespace"},
	{Name: "Instance"},
}

var supplyChainDataTypeByName = func() map[string]supplyChainDataType {
	out := make(map[string]supplyChainDataType, len(supplyChainDataTypes))
	for _, dt := range supplyChainDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
