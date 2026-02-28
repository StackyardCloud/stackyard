package server

type timestreamInfluxDBDataType struct {
	Name string
}

// Timestream for InfluxDB data types sourced from:
// https://docs.aws.amazon.com/ts-influxdb/latest/ts-influxdb-api/API_Types.html
var timestreamInfluxDBDataTypes = []timestreamInfluxDBDataType{
	{Name: "DbClusterSummary"},
	{Name: "DbInstanceForClusterSummary"},
	{Name: "DbInstanceSummary"},
	{Name: "DbParameterGroupSummary"},
	{Name: "Duration"},
	{Name: "InfluxDBv2Parameters"},
	{Name: "InfluxDBv3CoreParameters"},
	{Name: "InfluxDBv3EnterpriseParameters"},
	{Name: "LogDeliveryConfiguration"},
	{Name: "Parameters"},
	{Name: "PercentOrAbsoluteLong"},
	{Name: "S3Configuration"},
}

var timestreamInfluxDBDataTypeByName = func() map[string]timestreamInfluxDBDataType {
	out := make(map[string]timestreamInfluxDBDataType, len(timestreamInfluxDBDataTypes))
	for _, dataType := range timestreamInfluxDBDataTypes {
		out[dataType.Name] = dataType
	}
	return out
}()
