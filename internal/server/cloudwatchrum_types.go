package server

type cloudWatchRUMDataType struct {
	Name string
}

// Amazon CloudWatch RUM data types sourced from:
// https://docs.aws.amazon.com/cloudwatchrum/latest/APIReference/API_Types.html
var cloudWatchRUMDataTypes = []cloudWatchRUMDataType{
	{Name: "AppMonitor"},
	{Name: "AppMonitorConfiguration"},
	{Name: "AppMonitorDetails"},
	{Name: "AppMonitorSummary"},
	{Name: "BatchCreateRumMetricDefinitionsError"},
	{Name: "BatchDeleteRumMetricDefinitionsError"},
	{Name: "CustomEvents"},
	{Name: "CwLog"},
	{Name: "DataStorage"},
	{Name: "DeobfuscationConfiguration"},
	{Name: "JavaScriptSourceMaps"},
	{Name: "MetricDefinition"},
	{Name: "MetricDefinitionRequest"},
	{Name: "MetricDestinationSummary"},
	{Name: "QueryFilter"},
	{Name: "RumEvent"},
	{Name: "TimeRange"},
	{Name: "UserDetails"},
}

var cloudWatchRUMDataTypeByName = func() map[string]cloudWatchRUMDataType {
	out := make(map[string]cloudWatchRUMDataType, len(cloudWatchRUMDataTypes))
	for _, dt := range cloudWatchRUMDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
