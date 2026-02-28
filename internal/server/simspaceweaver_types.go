package server

type simSpaceWeaverDataType struct {
	Name string
}

// Amazon SimSpace Weaver data types sourced from:
// https://docs.aws.amazon.com/simspaceweaver/latest/APIReference/API_Types.html
var simSpaceWeaverDataTypes = []simSpaceWeaverDataType{
	{Name: "CloudWatchLogsLogGroup"},
	{Name: "Domain"},
	{Name: "LaunchOverrides"},
	{Name: "LiveSimulationState"},
	{Name: "LogDestination"},
	{Name: "LoggingConfiguration"},
	{Name: "S3Destination"},
	{Name: "S3Location"},
	{Name: "SimulationAppEndpointInfo"},
	{Name: "SimulationAppMetadata"},
	{Name: "SimulationAppPortMapping"},
	{Name: "SimulationClock"},
	{Name: "SimulationMetadata"},
}

var simSpaceWeaverDataTypeByName = func() map[string]simSpaceWeaverDataType {
	out := make(map[string]simSpaceWeaverDataType, len(simSpaceWeaverDataTypes))
	for _, dt := range simSpaceWeaverDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
