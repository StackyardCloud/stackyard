package server

type iotTwinMakerDataType struct {
	Name string
}

// AWS IoT TwinMaker data types sourced from:
// https://docs.aws.amazon.com/iot-twinmaker/latest/apireference/API_Types.html
var iotTwinMakerDataTypes = []iotTwinMakerDataType{
	{Name: "BatchPutPropertyError"},
	{Name: "BatchPutPropertyErrorEntry"},
	{Name: "BundleInformation"},
	{Name: "ColumnDescription"},
	{Name: "ComponentPropertyGroupRequest"},
	{Name: "ComponentPropertyGroupResponse"},
	{Name: "ComponentRequest"},
	{Name: "ComponentResponse"},
	{Name: "ComponentSummary"},
	{Name: "ComponentTypeSummary"},
	{Name: "ComponentUpdateRequest"},
	{Name: "CompositeComponentRequest"},
	{Name: "CompositeComponentTypeRequest"},
	{Name: "CompositeComponentTypeResponse"},
	{Name: "CompositeComponentUpdateRequest"},
	{Name: "DataConnector"},
	{Name: "DataType"},
	{Name: "DataValue"},
	{Name: "DestinationConfiguration"},
	{Name: "EntityPropertyReference"},
	{Name: "EntitySummary"},
	{Name: "ErrorDetails"},
	{Name: "FilterByAsset"},
	{Name: "FilterByAssetModel"},
	{Name: "FilterByComponentType"},
	{Name: "FilterByEntity"},
	{Name: "FunctionRequest"},
	{Name: "FunctionResponse"},
	{Name: "InterpolationParameters"},
	{Name: "IotSiteWiseSourceConfiguration"},
	{Name: "IotSiteWiseSourceConfigurationFilter"},
	{Name: "IotTwinMakerDestinationConfiguration"},
	{Name: "IotTwinMakerSourceConfiguration"},
	{Name: "IotTwinMakerSourceConfigurationFilter"},
	{Name: "LambdaFunction"},
	{Name: "ListComponentTypesFilter"},
	{Name: "ListEntitiesFilter"},
	{Name: "ListMetadataTransferJobsFilter"},
	{Name: "MetadataTransferJobProgress"},
	{Name: "MetadataTransferJobStatus"},
	{Name: "MetadataTransferJobSummary"},
	{Name: "OrderBy"},
	{Name: "ParentEntityUpdateRequest"},
	{Name: "PricingPlan"},
	{Name: "PropertyDefinitionRequest"},
	{Name: "PropertyDefinitionResponse"},
	{Name: "PropertyFilter"},
	{Name: "PropertyGroupRequest"},
	{Name: "PropertyGroupResponse"},
	{Name: "PropertyLatestValue"},
	{Name: "PropertyRequest"},
	{Name: "PropertyResponse"},
	{Name: "PropertySummary"},
	{Name: "PropertyValue"},
	{Name: "PropertyValueEntry"},
	{Name: "PropertyValueHistory"},
	{Name: "Relationship"},
	{Name: "RelationshipValue"},
	{Name: "Row"},
	{Name: "S3DestinationConfiguration"},
	{Name: "S3SourceConfiguration"},
	{Name: "SceneError"},
	{Name: "SceneSummary"},
	{Name: "SourceConfiguration"},
	{Name: "Status"},
	{Name: "SyncJobStatus"},
	{Name: "SyncJobSummary"},
	{Name: "SyncResourceFilter"},
	{Name: "SyncResourceStatus"},
	{Name: "SyncResourceSummary"},
	{Name: "TabularConditions"},
	{Name: "WorkspaceSummary"},
}

var iotTwinMakerDataTypeByName = func() map[string]iotTwinMakerDataType {
	out := make(map[string]iotTwinMakerDataType, len(iotTwinMakerDataTypes))
	for _, dt := range iotTwinMakerDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
