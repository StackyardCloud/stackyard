package server

type mwaaDataType struct {
	Name string
}

// Amazon Managed Workflows for Apache Airflow data types sourced from:
// https://docs.aws.amazon.com/mwaa/latest/API/API_Types.html
var mwaaDataTypes = []mwaaDataType{
	{Name: "Dimension"},
	{Name: "Environment"},
	{Name: "LastUpdate"},
	{Name: "LoggingConfiguration"},
	{Name: "LoggingConfigurationInput"},
	{Name: "MetricDatum"},
	{Name: "ModuleLoggingConfiguration"},
	{Name: "ModuleLoggingConfigurationInput"},
	{Name: "NetworkConfiguration"},
	{Name: "StatisticSet"},
	{Name: "UpdateEnvironment"},
	{Name: "UpdateError"},
	{Name: "UpdateNetworkConfigurationInput"},
}

var mwaaDataTypeByName = func() map[string]mwaaDataType {
	out := make(map[string]mwaaDataType, len(mwaaDataTypes))
	for _, dt := range mwaaDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
