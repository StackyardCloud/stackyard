package server

type gameliftStreamsDataType struct {
	Name string
}

// Amazon GameLift Streams API data types sourced from:
// https://docs.aws.amazon.com/gameliftstreams/latest/apireference/API_Types.html
var gameliftStreamsDataTypes = []gameliftStreamsDataType{
	{Name: "ApplicationSummary"},
	{Name: "DefaultApplication"},
	{Name: "ExportFilesMetadata"},
	{Name: "LocationConfiguration"},
	{Name: "LocationState"},
	{Name: "PerformanceStatsConfiguration"},
	{Name: "ReplicationStatus"},
	{Name: "RuntimeEnvironment"},
	{Name: "StreamGroupSummary"},
	{Name: "StreamSessionSummary"},
}

var gameliftStreamsDataTypeByName = func() map[string]gameliftStreamsDataType {
	out := make(map[string]gameliftStreamsDataType, len(gameliftStreamsDataTypes))
	for _, dt := range gameliftStreamsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
