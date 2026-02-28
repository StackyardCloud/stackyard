package server

type networkMonitorDataType struct {
	Name string
}

// Amazon Network Synthetic Monitor data types sourced from:
// https://docs.aws.amazon.com/networkmonitor/latest/APIReference/API_Types.html
var networkMonitorDataTypes = []networkMonitorDataType{
	{Name: "CreateMonitorProbeInput"},
	{Name: "MonitorSummary"},
	{Name: "Probe"},
	{Name: "ProbeInput"},
	{Name: "UpdateProbe"},
}

var networkMonitorDataTypeByName = func() map[string]networkMonitorDataType {
	out := make(map[string]networkMonitorDataType, len(networkMonitorDataTypes))
	for _, dt := range networkMonitorDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
