package server

type deviceFarmDataType struct {
	Name string
}

// AWS Device Farm data types sourced from:
// https://docs.aws.amazon.com/devicefarm/latest/APIReference/API_Types.html
var deviceFarmDataTypes = []deviceFarmDataType{
	{Name: "AccountSettings"},
	{Name: "Artifact"},
	{Name: "CPU"},
	{Name: "Counters"},
	{Name: "CreateRemoteAccessSessionConfiguration"},
	{Name: "CustomerArtifactPaths"},
	{Name: "Device"},
	{Name: "DeviceFilter"},
	{Name: "DeviceInstance"},
	{Name: "DeviceMinutes"},
	{Name: "DevicePool"},
	{Name: "DevicePoolCompatibilityResult"},
	{Name: "DeviceProxy"},
	{Name: "DeviceSelectionConfiguration"},
	{Name: "DeviceSelectionResult"},
	{Name: "EnvironmentVariable"},
	{Name: "ExecutionConfiguration"},
	{Name: "IncompatibilityMessage"},
	{Name: "InstanceProfile"},
	{Name: "Job"},
	{Name: "ListTestGridProjectsRequest"},
	{Name: "ListTestGridSessionsRequest"},
	{Name: "Location"},
	{Name: "MonetaryAmount"},
	{Name: "NetworkProfile"},
	{Name: "Offering"},
	{Name: "OfferingPromotion"},
	{Name: "OfferingStatus"},
	{Name: "OfferingTransaction"},
	{Name: "Problem"},
	{Name: "ProblemDetail"},
	{Name: "Project"},
	{Name: "Radios"},
	{Name: "RecurringCharge"},
	{Name: "RemoteAccessEndpoints"},
	{Name: "RemoteAccessSession"},
	{Name: "Resolution"},
	{Name: "Rule"},
	{Name: "Run"},
	{Name: "Sample"},
	{Name: "ScheduleRunConfiguration"},
	{Name: "ScheduleRunTest"},
	{Name: "Suite"},
	{Name: "Tag"},
	{Name: "Test"},
	{Name: "TestGridProject"},
	{Name: "TestGridSession"},
	{Name: "TestGridSessionAction"},
	{Name: "TestGridSessionArtifact"},
	{Name: "TestGridVpcConfig"},
	{Name: "TrialMinutes"},
	{Name: "UniqueProblem"},
	{Name: "UpdateVPCEConfiguration"},
	{Name: "Upload"},
	{Name: "VPCEConfiguration"},
	{Name: "VpcConfig"},
}

var deviceFarmDataTypeByName = func() map[string]deviceFarmDataType {
	out := make(map[string]deviceFarmDataType, len(deviceFarmDataTypes))
	for _, dt := range deviceFarmDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
