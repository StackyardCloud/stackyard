package server

type iotFleetWiseDataType struct {
	Name string
}

// AWS IoT FleetWise data types sourced from:
// https://docs.aws.amazon.com/iot-fleetwise/latest/APIReference/API_Types.html
var iotFleetWiseDataTypes = []iotFleetWiseDataType{
	{Name: "Actuator"},
	{Name: "Attribute"},
	{Name: "Branch"},
	{Name: "CampaignSummary"},
	{Name: "CanDbcDefinition"},
	{Name: "CanInterface"},
	{Name: "CanSignal"},
	{Name: "CloudWatchLogDeliveryOptions"},
	{Name: "CollectionScheme"},
	{Name: "ConditionBasedCollectionScheme"},
	{Name: "ConditionBasedSignalFetchConfig"},
	{Name: "CreateVehicleError"},
	{Name: "CreateVehicleRequestItem"},
	{Name: "CreateVehicleResponseItem"},
	{Name: "CustomDecodingInterface"},
	{Name: "CustomDecodingSignal"},
	{Name: "CustomProperty"},
	{Name: "CustomStruct"},
	{Name: "DataDestinationConfig"},
	{Name: "DataPartition"},
	{Name: "DataPartitionStorageOptions"},
	{Name: "DataPartitionUploadOptions"},
	{Name: "DecoderManifestSummary"},
	{Name: "FleetSummary"},
	{Name: "FormattedVss"},
	{Name: "IamRegistrationResponse"},
	{Name: "IamResources"},
	{Name: "InvalidNetworkInterface"},
	{Name: "InvalidSignal"},
	{Name: "InvalidSignalDecoder"},
	{Name: "MessageSignal"},
	{Name: "ModelManifestSummary"},
	{Name: "MqttTopicConfig"},
	{Name: "NetworkFileDefinition"},
	{Name: "NetworkInterface"},
	{Name: "Node"},
	{Name: "NodeCounts"},
	{Name: "ObdInterface"},
	{Name: "ObdSignal"},
	{Name: "OnChangeStateTemplateUpdateStrategy"},
	{Name: "PeriodicStateTemplateUpdateStrategy"},
	{Name: "PrimitiveMessageDefinition"},
	{Name: "ROS2PrimitiveMessageDefinition"},
	{Name: "S3Config"},
	{Name: "Sensor"},
	{Name: "SignalCatalogSummary"},
	{Name: "SignalDecoder"},
	{Name: "SignalFetchConfig"},
	{Name: "SignalFetchInformation"},
	{Name: "SignalInformation"},
	{Name: "StateTemplateAssociation"},
	{Name: "StateTemplateSummary"},
	{Name: "StateTemplateUpdateStrategy"},
	{Name: "StorageMaximumSize"},
	{Name: "StorageMinimumTimeToLive"},
	{Name: "StructuredMessage"},
	{Name: "StructuredMessageFieldNameAndDataTypePair"},
	{Name: "StructuredMessageListDefinition"},
	{Name: "Tag"},
	{Name: "TimeBasedCollectionScheme"},
	{Name: "TimeBasedSignalFetchConfig"},
	{Name: "TimePeriod"},
	{Name: "TimestreamConfig"},
	{Name: "TimestreamRegistrationResponse"},
	{Name: "TimestreamResources"},
	{Name: "UpdateVehicleError"},
	{Name: "UpdateVehicleRequestItem"},
	{Name: "UpdateVehicleResponseItem"},
	{Name: "ValidationExceptionField"},
	{Name: "VehicleMiddleware"},
	{Name: "VehicleStatus"},
	{Name: "VehicleSummary"},
}

var iotFleetWiseDataTypeByName = func() map[string]iotFleetWiseDataType {
	out := make(map[string]iotFleetWiseDataType, len(iotFleetWiseDataTypes))
	for _, t := range iotFleetWiseDataTypes {
		out[t.Name] = t
	}
	return out
}()
