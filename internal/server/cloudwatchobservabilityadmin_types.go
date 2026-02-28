package server

type cloudWatchObservabilityAdminDataType struct {
	Name string
}

// Amazon CloudWatch Observability Admin data types sourced from:
// https://docs.aws.amazon.com/cloudwatch/latest/observabilityadmin/API_Types.html
var cloudWatchObservabilityAdminDataTypes = []cloudWatchObservabilityAdminDataType{
	{Name: "ActionCondition"},
	{Name: "AdvancedEventSelector"},
	{Name: "AdvancedFieldSelector"},
	{Name: "CentralizationRule"},
	{Name: "CentralizationRuleDestination"},
	{Name: "CentralizationRuleSource"},
	{Name: "CentralizationRuleSummary"},
	{Name: "CloudtrailParameters"},
	{Name: "Condition"},
	{Name: "ConfigurationSummary"},
	{Name: "DataSource"},
	{Name: "DestinationLogsConfiguration"},
	{Name: "ELBLoadBalancerLoggingParameters"},
	{Name: "Encryption"},
	{Name: "FieldToMatch"},
	{Name: "Filter"},
	{Name: "IntegrationSummary"},
	{Name: "LabelNameCondition"},
	{Name: "LogDeliveryParameters"},
	{Name: "LoggingFilter"},
	{Name: "LogsBackupConfiguration"},
	{Name: "LogsEncryptionConfiguration"},
	{Name: "PipelineOutput"},
	{Name: "PipelineOutputError"},
	{Name: "Record"},
	{Name: "SingleHeader"},
	{Name: "Source"},
	{Name: "SourceLogsConfiguration"},
	{Name: "TelemetryConfiguration"},
	{Name: "TelemetryDestinationConfiguration"},
	{Name: "TelemetryPipeline"},
	{Name: "TelemetryPipelineConfiguration"},
	{Name: "TelemetryPipelineStatusReason"},
	{Name: "TelemetryPipelineSummary"},
	{Name: "TelemetryRule"},
	{Name: "TelemetryRuleSummary"},
	{Name: "VPCFlowLogParameters"},
	{Name: "ValidateTelemetryPipelineConfiguration"},
	{Name: "ValidationError"},
	{Name: "WAFLoggingParameters"},
}

var cloudWatchObservabilityAdminDataTypeByName = func() map[string]cloudWatchObservabilityAdminDataType {
	out := make(map[string]cloudWatchObservabilityAdminDataType, len(cloudWatchObservabilityAdminDataTypes))
	for _, dt := range cloudWatchObservabilityAdminDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
