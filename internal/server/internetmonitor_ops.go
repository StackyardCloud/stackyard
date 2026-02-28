package server

type internetMonitorOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Internet Monitor operations sourced from:
// https://docs.aws.amazon.com/internet-monitor/latest/api/API_Operations.html
var internetMonitorOperations = []internetMonitorOperation{
	{Name: "CreateMonitor", Method: "POST", URI: "/v20210603/Monitors"},
	{Name: "DeleteMonitor", Method: "DELETE", URI: "/v20210603/Monitors/{MonitorName}"},
	{Name: "GetHealthEvent", Method: "GET", URI: "/v20210603/Monitors/{MonitorName}/HealthEvents/{EventId}"},
	{Name: "GetInternetEvent", Method: "GET", URI: "/v20210603/InternetEvents/{EventId}"},
	{Name: "GetMonitor", Method: "GET", URI: "/v20210603/Monitors/{MonitorName}"},
	{Name: "GetQueryResults", Method: "GET", URI: "/v20210603/Monitors/{MonitorName}/Queries/{QueryId}/Results"},
	{Name: "GetQueryStatus", Method: "GET", URI: "/v20210603/Monitors/{MonitorName}/Queries/{QueryId}/Status"},
	{Name: "ListHealthEvents", Method: "GET", URI: "/v20210603/Monitors/{MonitorName}/HealthEvents"},
	{Name: "ListInternetEvents", Method: "GET", URI: "/v20210603/InternetEvents"},
	{Name: "ListMonitors", Method: "GET", URI: "/v20210603/Monitors"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "StartQuery", Method: "POST", URI: "/v20210603/Monitors/{MonitorName}/Queries"},
	{Name: "StopQuery", Method: "DELETE", URI: "/v20210603/Monitors/{MonitorName}/Queries/{QueryId}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateMonitor", Method: "PATCH", URI: "/v20210603/Monitors/{MonitorName}"},
}

var internetMonitorOperationByName = func() map[string]internetMonitorOperation {
	out := make(map[string]internetMonitorOperation, len(internetMonitorOperations))
	for _, op := range internetMonitorOperations {
		out[op.Name] = op
	}
	return out
}()
