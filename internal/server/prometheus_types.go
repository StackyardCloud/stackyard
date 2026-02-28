package server

type prometheusDataType struct {
	Name string
}

// Amazon Managed Service for Prometheus data types sourced from:
// https://docs.aws.amazon.com/prometheus/latest/APIReference/API_Types.html
var prometheusDataTypes = []prometheusDataType{
	{Name: "AlertManagerDefinitionDescription"},
	{Name: "AlertManagerDefinitionStatus"},
	{Name: "AmpConfiguration"},
	{Name: "AnomalyDetectorConfiguration"},
	{Name: "AnomalyDetectorDescription"},
	{Name: "AnomalyDetectorMissingDataAction"},
	{Name: "AnomalyDetectorStatus"},
	{Name: "AnomalyDetectorSummary"},
	{Name: "CloudWatchLogDestination"},
	{Name: "ComponentConfig"},
	{Name: "Destination"},
	{Name: "EksConfiguration"},
	{Name: "IgnoreNearExpected"},
	{Name: "LimitsPerLabelSet"},
	{Name: "LimitsPerLabelSetEntry"},
	{Name: "LoggingConfigurationMetadata"},
	{Name: "LoggingConfigurationStatus"},
	{Name: "LoggingDestination"},
	{Name: "LoggingFilter"},
	{Name: "QueryLoggingConfigurationMetadata"},
	{Name: "QueryLoggingConfigurationStatus"},
	{Name: "RandomCutForestConfiguration"},
	{Name: "RoleConfiguration"},
	{Name: "RuleGroupsNamespaceDescription"},
	{Name: "RuleGroupsNamespaceStatus"},
	{Name: "RuleGroupsNamespaceSummary"},
	{Name: "ScrapeConfiguration"},
	{Name: "ScraperComponent"},
	{Name: "ScraperDescription"},
	{Name: "ScraperLoggingConfigurationStatus"},
	{Name: "ScraperLoggingDestination"},
	{Name: "ScraperStatus"},
	{Name: "ScraperSummary"},
	{Name: "Source"},
	{Name: "UpdateWorkspaceConfiguration"},
	{Name: "ValidationExceptionField"},
	{Name: "VpcConfiguration"},
	{Name: "WorkspaceConfigurationDescription"},
	{Name: "WorkspaceConfigurationStatus"},
	{Name: "WorkspaceDescription"},
	{Name: "WorkspaceStatus"},
	{Name: "WorkspaceSummary"},
}

var prometheusDataTypeByName = func() map[string]prometheusDataType {
	out := make(map[string]prometheusDataType, len(prometheusDataTypes))
	for _, dt := range prometheusDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
