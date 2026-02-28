package server

type mwaaOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Managed Workflows for Apache Airflow actions sourced from:
// https://docs.aws.amazon.com/mwaa/latest/API/API_Operations.html
var mwaaOperations = []mwaaOperation{
	{Name: "CreateCliToken", Method: "POST", URI: "/clitoken/{Name}"},
	{Name: "CreateEnvironment", Method: "PUT", URI: "/environments/{Name}"},
	{Name: "CreateWebLoginToken", Method: "POST", URI: "/webtoken/{Name}"},
	{Name: "DeleteEnvironment", Method: "DELETE", URI: "/environments/{Name}"},
	{Name: "GetEnvironment", Method: "GET", URI: "/environments/{Name}"},
	{Name: "InvokeRestApi", Method: "POST", URI: "/restapi/{Name}"},
	{Name: "ListEnvironments", Method: "GET", URI: "/environments?MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "PublishMetrics", Method: "POST", URI: "/metrics/environments/{EnvironmentName}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateEnvironment", Method: "PATCH", URI: "/environments/{Name}"},
}

var mwaaOperationByName = func() map[string]mwaaOperation {
	out := make(map[string]mwaaOperation, len(mwaaOperations))
	for _, op := range mwaaOperations {
		out[op.Name] = op
	}
	return out
}()
