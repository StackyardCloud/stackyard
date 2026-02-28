package server

type iotMIDataType struct {
	Name string
}

// AWS IoT Managed Integrations data types sourced from:
// https://docs.aws.amazon.com/iot-mi/latest/APIReference/API_Types.html
var iotMIDataTypes = []iotMIDataType{
	{Name: "AbortConfigCriteria"},
	{Name: "AccountAssociationItem"},
	{Name: "AuthConfig"},
	{Name: "AuthConfigUpdate"},
	{Name: "AuthMaterial"},
	{Name: "CapabilityAction"},
	{Name: "CapabilityReport"},
	{Name: "CapabilityReportCapability"},
	{Name: "CapabilityReportEndpoint"},
	{Name: "CapabilitySchemaItem"},
	{Name: "CommandCapability"},
	{Name: "CommandEndpoint"},
	{Name: "ConfigurationError"},
	{Name: "ConfigurationStatus"},
	{Name: "ConnectorDestinationSummary"},
	{Name: "ConnectorItem"},
	{Name: "CredentialLockerSummary"},
	{Name: "DestinationSummary"},
	{Name: "Device"},
	{Name: "DeviceDiscoverySummary"},
	{Name: "DiscoveredDeviceSummary"},
	{Name: "EndpointConfig"},
	{Name: "EventLogConfigurationSummary"},
	{Name: "ExponentialRolloutRate"},
	{Name: "GeneralAuthorizationName"},
	{Name: "GeneralAuthorizationUpdate"},
	{Name: "LambdaConfig"},
	{Name: "ManagedThingAssociation"},
	{Name: "ManagedThingSchemaListItem"},
	{Name: "ManagedThingSummary"},
	{Name: "MatterCapabilityReport"},
	{Name: "MatterCapabilityReportAttribute"},
	{Name: "MatterCapabilityReportCluster"},
	{Name: "MatterCapabilityReportEndpoint"},
	{Name: "MatterCluster"},
	{Name: "MatterEndpoint"},
	{Name: "NotificationConfigurationSummary"},
	{Name: "OAuthConfig"},
	{Name: "OAuthUpdate"},
	{Name: "OtaTaskAbortConfig"},
	{Name: "OtaTaskConfigurationSummary"},
	{Name: "OtaTaskExecutionRetryConfig"},
	{Name: "OtaTaskExecutionRolloutConfig"},
	{Name: "OtaTaskExecutionSummaries"},
	{Name: "OtaTaskExecutionSummary"},
	{Name: "OtaTaskSchedulingConfig"},
	{Name: "OtaTaskSummary"},
	{Name: "OtaTaskTimeoutConfig"},
	{Name: "ProactiveRefreshTokenRenewal"},
	{Name: "ProvisioningProfileSummary"},
	{Name: "PushConfig"},
	{Name: "RetryConfigCriteria"},
	{Name: "RolloutRateIncreaseCriteria"},
	{Name: "RuntimeLogConfigurations"},
	{Name: "ScheduleMaintenanceWindow"},
	{Name: "SchemaVersionListItem"},
	{Name: "SecretsManager"},
	{Name: "StateCapability"},
	{Name: "StateEndpoint"},
	{Name: "TaskProcessingDetails"},
	{Name: "WiFiSimpleSetupConfiguration"},
	{Name: "UpdateOtaTask"},
}

var iotMIDataTypeByName = func() map[string]iotMIDataType {
	out := make(map[string]iotMIDataType, len(iotMIDataTypes))
	for _, dataType := range iotMIDataTypes {
		out[dataType.Name] = dataType
	}
	return out
}()
