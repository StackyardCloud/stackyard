package server

type neptuneAnalyticsDataType struct {
	Name string
}

// Neptune Analytics API data types sourced from:
// https://docs.aws.amazon.com/neptune-analytics/latest/apiref/API_Types.html
var neptuneAnalyticsDataTypes = []neptuneAnalyticsDataType{
	{Name: "EdgeStructure"},
	{Name: "ExportFilter"},
	{Name: "ExportFilterElement"},
	{Name: "ExportFilterPropertyAttributes"},
	{Name: "ExportTaskDetails"},
	{Name: "ExportTaskSummary"},
	{Name: "GraphDataSummary"},
	{Name: "GraphSnapshotSummary"},
	{Name: "GraphSummary"},
	{Name: "ImportOptions"},
	{Name: "ImportTaskDetails"},
	{Name: "ImportTaskSummary"},
	{Name: "NeptuneImportOptions"},
	{Name: "NodeStructure"},
	{Name: "PrivateGraphEndpointSummary"},
	{Name: "QuerySummary"},
	{Name: "VectorSearchConfiguration"},
}

var neptuneAnalyticsDataTypeByName = func() map[string]neptuneAnalyticsDataType {
	out := make(map[string]neptuneAnalyticsDataType, len(neptuneAnalyticsDataTypes))
	for _, dt := range neptuneAnalyticsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
