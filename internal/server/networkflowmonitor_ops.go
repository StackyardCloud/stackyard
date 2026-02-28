package server

type networkFlowMonitorOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Network Flow Monitor operations sourced from:
// https://docs.aws.amazon.com/networkflowmonitor/2.0/APIReference/API_Operations.html
var networkFlowMonitorOperations = []networkFlowMonitorOperation{
	{Name: "CreateMonitor", Method: "POST", URI: "/monitors"},
	{Name: "CreateScope", Method: "POST", URI: "/scopes"},
	{Name: "DeleteMonitor", Method: "DELETE", URI: "/monitors/{monitorName}"},
	{Name: "DeleteScope", Method: "DELETE", URI: "/scopes/{scopeId}"},
	{Name: "GetMonitor", Method: "GET", URI: "/monitors/{monitorName}"},
	{Name: "GetQueryResultsMonitorTopContributors", Method: "GET", URI: "/monitors/{monitorName}/topContributorsQueries/{queryId}/results"},
	{Name: "GetQueryResultsWorkloadInsightsTopContributors", Method: "GET", URI: "/workloadInsights/{scopeId}/topContributorsQueries/{queryId}/results"},
	{Name: "GetQueryResultsWorkloadInsightsTopContributorsData", Method: "GET", URI: "/workloadInsights/{scopeId}/topContributorsDataQueries/{queryId}/results"},
	{Name: "GetQueryStatusMonitorTopContributors", Method: "GET", URI: "/monitors/{monitorName}/topContributorsQueries/{queryId}/status"},
	{Name: "GetQueryStatusWorkloadInsightsTopContributors", Method: "GET", URI: "/workloadInsights/{scopeId}/topContributorsQueries/{queryId}/status"},
	{Name: "GetQueryStatusWorkloadInsightsTopContributorsData", Method: "GET", URI: "/workloadInsights/{scopeId}/topContributorsDataQueries/{queryId}/status"},
	{Name: "GetScope", Method: "GET", URI: "/scopes/{scopeId}"},
	{Name: "ListMonitors", Method: "GET", URI: "/monitors"},
	{Name: "ListScopes", Method: "GET", URI: "/scopes"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "StartQueryMonitorTopContributors", Method: "POST", URI: "/monitors/{monitorName}/topContributorsQueries"},
	{Name: "StartQueryWorkloadInsightsTopContributors", Method: "POST", URI: "/workloadInsights/{scopeId}/topContributorsQueries"},
	{Name: "StartQueryWorkloadInsightsTopContributorsData", Method: "POST", URI: "/workloadInsights/{scopeId}/topContributorsDataQueries"},
	{Name: "StopQueryMonitorTopContributors", Method: "DELETE", URI: "/monitors/{monitorName}/topContributorsQueries/{queryId}"},
	{Name: "StopQueryWorkloadInsightsTopContributors", Method: "DELETE", URI: "/workloadInsights/{scopeId}/topContributorsQueries/{queryId}"},
	{Name: "StopQueryWorkloadInsightsTopContributorsData", Method: "DELETE", URI: "/workloadInsights/{scopeId}/topContributorsDataQueries/{queryId}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateMonitor", Method: "PATCH", URI: "/monitors/{monitorName}"},
	{Name: "UpdateScope", Method: "PATCH", URI: "/scopes/{scopeId}"},
}

var networkFlowMonitorOperationByName = func() map[string]networkFlowMonitorOperation {
	out := make(map[string]networkFlowMonitorOperation, len(networkFlowMonitorOperations))
	for _, op := range networkFlowMonitorOperations {
		out[op.Name] = op
	}
	return out
}()
