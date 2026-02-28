package server

type networkFlowMonitorDataType struct {
	Name string
}

// Amazon Network Flow Monitor data types sourced from:
// https://docs.aws.amazon.com/networkflowmonitor/2.0/APIReference/API_Types.html
var networkFlowMonitorDataTypes = []networkFlowMonitorDataType{
	{Name: "KubernetesMetadata"},
	{Name: "MonitorLocalResource"},
	{Name: "MonitorRemoteResource"},
	{Name: "MonitorSummary"},
	{Name: "MonitorTopContributorsRow"},
	{Name: "ScopeSummary"},
	{Name: "TargetId"},
	{Name: "TargetIdentifier"},
	{Name: "TargetResource"},
	{Name: "TraversedComponent"},
	{Name: "WorkloadInsightsTopContributorsDataPoint"},
	{Name: "WorkloadInsightsTopContributorsRow"},
}

var networkFlowMonitorDataTypeByName = func() map[string]networkFlowMonitorDataType {
	out := make(map[string]networkFlowMonitorDataType, len(networkFlowMonitorDataTypes))
	for _, dt := range networkFlowMonitorDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
