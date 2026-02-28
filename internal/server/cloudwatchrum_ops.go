package server

type cloudWatchRUMOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon CloudWatch RUM operations sourced from:
// https://docs.aws.amazon.com/cloudwatchrum/latest/APIReference/API_Operations.html
var cloudWatchRUMOperations = []cloudWatchRUMOperation{
	{Name: "BatchCreateRumMetricDefinitions", Method: "POST", URI: "/rummetrics/{AppMonitorName}"},
	{Name: "BatchDeleteRumMetricDefinitions", Method: "DELETE", URI: "/rummetrics/{AppMonitorName}"},
	{Name: "BatchGetRumMetricDefinitions", Method: "GET", URI: "/rummetrics/{AppMonitorName}"},
	{Name: "CreateAppMonitor", Method: "POST", URI: "/appmonitor"},
	{Name: "DeleteAppMonitor", Method: "DELETE", URI: "/appmonitor/{Name}"},
	{Name: "DeleteResourcePolicy", Method: "DELETE", URI: "/appmonitor/{Name}"},
	{Name: "DeleteRumMetricsDestination", Method: "DELETE", URI: "/rummetrics/{AppMonitorName}"},
	{Name: "GetAppMonitor", Method: "GET", URI: "/appmonitor/{Name}"},
	{Name: "GetAppMonitorData", Method: "POST", URI: "/appmonitor/{Name}"},
	{Name: "GetResourcePolicy", Method: "GET", URI: "/appmonitor/{Name}"},
	{Name: "ListAppMonitors", Method: "POST", URI: "/appmonitors"},
	{Name: "ListRumMetricsDestinations", Method: "GET", URI: "/rummetrics/{AppMonitorName}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "PutResourcePolicy", Method: "PUT", URI: "/appmonitor/{Name}"},
	{Name: "PutRumEvents", Method: "POST", URI: "/appmonitors/{Id}"},
	{Name: "PutRumMetricsDestination", Method: "POST", URI: "/rummetrics/{AppMonitorName}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateAppMonitor", Method: "PATCH", URI: "/appmonitor/{Name}"},
	{Name: "UpdateRumMetricDefinition", Method: "PATCH", URI: "/rummetrics/{AppMonitorName}"},
}

var cloudWatchRUMOperationByName = func() map[string]cloudWatchRUMOperation {
	out := make(map[string]cloudWatchRUMOperation, len(cloudWatchRUMOperations))
	for _, op := range cloudWatchRUMOperations {
		out[op.Name] = op
	}
	return out
}()
