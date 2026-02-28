package server

type emrServerlessOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon EMR Serverless actions sourced from:
// https://docs.aws.amazon.com/emr-serverless/latest/APIReference/API_Operations.html
var emrServerlessOperations = []emrServerlessOperation{
	{Name: "CancelJobRun", Method: "DELETE", URI: "/applications/{applicationId}/jobruns/{jobRunId}?shutdownGracePeriodInSeconds={shutdownGracePeriodInSeconds}"},
	{Name: "CreateApplication", Method: "POST", URI: "/applications"},
	{Name: "DeleteApplication", Method: "DELETE", URI: "/applications/{applicationId}"},
	{Name: "GetApplication", Method: "GET", URI: "/applications/{applicationId}"},
	{Name: "GetDashboardForJobRun", Method: "GET", URI: "/applications/{applicationId}/jobruns/{jobRunId}/dashboard?accessSystemProfileLogs={accessSystemProfileLogs}&attempt={attempt}"},
	{Name: "GetJobRun", Method: "GET", URI: "/applications/{applicationId}/jobruns/{jobRunId}?attempt={attempt}"},
	{Name: "ListApplications", Method: "GET", URI: "/applications?maxResults={maxResults}&nextToken={nextToken}&states={states}"},
	{Name: "ListJobRunAttempts", Method: "GET", URI: "/applications/{applicationId}/jobruns/{jobRunId}/attempts?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListJobRuns", Method: "GET", URI: "/applications/{applicationId}/jobruns?createdAtAfter={createdAtAfter}&createdAtBefore={createdAtBefore}&maxResults={maxResults}&mode={mode}&nextToken={nextToken}&states={states}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "StartApplication", Method: "POST", URI: "/applications/{applicationId}/start"},
	{Name: "StartJobRun", Method: "POST", URI: "/applications/{applicationId}/jobruns"},
	{Name: "StopApplication", Method: "POST", URI: "/applications/{applicationId}/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateApplication", Method: "PATCH", URI: "/applications/{applicationId}"},
}

var emrServerlessOperationByName = func() map[string]emrServerlessOperation {
	out := make(map[string]emrServerlessOperation, len(emrServerlessOperations))
	for _, op := range emrServerlessOperations {
		out[op.Name] = op
	}
	return out
}()
