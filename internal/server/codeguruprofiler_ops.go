package server

type codeGuruProfilerOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon CodeGuru Profiler operations sourced from:
// https://docs.aws.amazon.com/codeguru/latest/profiler-api/API_Operations.html
var codeGuruProfilerOperations = []codeGuruProfilerOperation{
	{Name: "AddNotificationChannels", Method: "POST", URI: "/profilingGroups/{profilingGroupName}/notificationConfiguration"},
	{Name: "BatchGetFrameMetricData", Method: "POST", URI: "/profilingGroups/{profilingGroupName}/frames/-/metrics"},
	{Name: "ConfigureAgent", Method: "POST", URI: "/profilingGroups/{profilingGroupName}/configureAgent"},
	{Name: "CreateProfilingGroup", Method: "POST", URI: "/profilingGroups"},
	{Name: "DeleteProfilingGroup", Method: "DELETE", URI: "/profilingGroups/{profilingGroupName}"},
	{Name: "DescribeProfilingGroup", Method: "GET", URI: "/profilingGroups/{profilingGroupName}"},
	{Name: "GetFindingsReportAccountSummary", Method: "GET", URI: "/internal/findingsReports"},
	{Name: "GetNotificationConfiguration", Method: "GET", URI: "/profilingGroups/{profilingGroupName}/notificationConfiguration"},
	{Name: "GetPolicy", Method: "GET", URI: "/profilingGroups/{profilingGroupName}/policy"},
	{Name: "GetProfile", Method: "GET", URI: "/profilingGroups/{profilingGroupName}/profile"},
	{Name: "GetRecommendations", Method: "GET", URI: "/internal/profilingGroups/{profilingGroupName}/recommendations"},
	{Name: "ListFindingsReports", Method: "GET", URI: "/internal/profilingGroups/{profilingGroupName}/findingsReports"},
	{Name: "ListProfileTimes", Method: "GET", URI: "/profilingGroups/{profilingGroupName}/profileTimes"},
	{Name: "ListProfilingGroups", Method: "GET", URI: "/profilingGroups"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "PostAgentProfile", Method: "POST", URI: "/profilingGroups/{profilingGroupName}/agentProfile"},
	{Name: "PutPermission", Method: "PUT", URI: "/profilingGroups/{profilingGroupName}/policy/{actionGroup}"},
	{Name: "RemoveNotificationChannel", Method: "DELETE", URI: "/profilingGroups/{profilingGroupName}/notificationConfiguration/{channelId}"},
	{Name: "RemovePermission", Method: "DELETE", URI: "/profilingGroups/{profilingGroupName}/policy/{actionGroup}"},
	{Name: "SubmitFeedback", Method: "POST", URI: "/internal/profilingGroups/{profilingGroupName}/anomalies/{anomalyInstanceId}/feedback"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateProfilingGroup", Method: "PUT", URI: "/profilingGroups/{profilingGroupName}"},
}

var codeGuruProfilerOperationByName = func() map[string]codeGuruProfilerOperation {
	out := make(map[string]codeGuruProfilerOperation, len(codeGuruProfilerOperations))
	for _, op := range codeGuruProfilerOperations {
		out[op.Name] = op
	}
	return out
}()
