package server

type networkMonitorOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Network Synthetic Monitor operations sourced from:
// https://docs.aws.amazon.com/networkmonitor/latest/APIReference/API_Operations.html
var networkMonitorOperations = []networkMonitorOperation{
	{Name: "CreateMonitor", Method: "POST", URI: "/monitors"},
	{Name: "CreateProbe", Method: "POST", URI: "/monitors/{monitorName}/probes"},
	{Name: "DeleteMonitor", Method: "DELETE", URI: "/monitors/{monitorName}"},
	{Name: "DeleteProbe", Method: "DELETE", URI: "/monitors/{monitorName}/probes/{probeId}"},
	{Name: "GetMonitor", Method: "GET", URI: "/monitors/{monitorName}"},
	{Name: "GetProbe", Method: "GET", URI: "/monitors/{monitorName}/probes/{probeId}"},
	{Name: "ListMonitors", Method: "GET", URI: "/monitors"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateMonitor", Method: "PATCH", URI: "/monitors/{monitorName}"},
	{Name: "UpdateProbe", Method: "PATCH", URI: "/monitors/{monitorName}/probes/{probeId}"},
}

var networkMonitorOperationByName = func() map[string]networkMonitorOperation {
	out := make(map[string]networkMonitorOperation, len(networkMonitorOperations))
	for _, op := range networkMonitorOperations {
		out[op.Name] = op
	}
	return out
}()
