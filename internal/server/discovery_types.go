package server

type discoveryDataType struct {
	Name string
}

// AWS Application Discovery Service data types sourced from:
// https://docs.aws.amazon.com/application-discovery/latest/APIReference/API_Types.html
var discoveryDataTypes = []discoveryDataType{
	{Name: "AgentConfigurationStatus"},
	{Name: "AgentInfo"},
	{Name: "AgentNetworkInfo"},
	{Name: "BatchDeleteAgentError"},
	{Name: "BatchDeleteConfigurationTask"},
	{Name: "BatchDeleteImportDataError"},
	{Name: "ConfigurationTag"},
	{Name: "ContinuousExportDescription"},
	{Name: "CustomerAgentInfo"},
	{Name: "CustomerAgentlessCollectorInfo"},
	{Name: "CustomerConnectorInfo"},
	{Name: "CustomerMeCollectorInfo"},
	{Name: "DeleteAgent"},
	{Name: "DeletionWarning"},
	{Name: "Ec2RecommendationsExportPreferences"},
	{Name: "ExportFilter"},
	{Name: "ExportInfo"},
	{Name: "ExportPreferences"},
	{Name: "FailedConfiguration"},
	{Name: "Filter"},
	{Name: "ImportTask"},
	{Name: "ImportTaskFilter"},
	{Name: "NeighborConnectionDetail"},
	{Name: "OrderByElement"},
	{Name: "ReservedInstanceOptions"},
	{Name: "Tag"},
	{Name: "TagFilter"},
	{Name: "UpdateApplication"},
	{Name: "UsageMetricBasis"},
}

var discoveryDataTypeByName = func() map[string]discoveryDataType {
	out := make(map[string]discoveryDataType, len(discoveryDataTypes))
	for _, dt := range discoveryDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
